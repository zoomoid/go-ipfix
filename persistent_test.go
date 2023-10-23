package ipfix

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"testing"
)

func cacheFactory(file *os.File) (StatefulTemplateCache, error) {
	underlyingTemplateCache := NewNamedEphemeralCache("backing_cache")

	// this field cache does not load any field definitions, we need to add them manually
	// for a field manager with loaded IPFIX fields, see pkg/collector/internal/managers/fields.go
	fieldManager := NewEphemeralFieldCache(underlyingTemplateCache)

	for id, f := range IPFIX() {
		if f.Id == 0 {
			f.Id = id
		}
		err := fieldManager.Add(context.Background(), f)
		if err != nil {
			return nil, err
		}
	}

	cache := NewNamedPersistentCache("persistence_test", file, fieldManager, underlyingTemplateCache)

	return cache, nil
}

func TestPersistentCache(t *testing.T) {
	// dir := t.TempDir()

	t.Run("without restore", func(t *testing.T) {
		// this is a fresh file, there is nothing to initialize from currently
		p := path.Join(".", "fixture_persistent_test_without_restore.json")
		file, err := os.Open(p)
		if err != nil {
			file, err = os.Create(p)
			if err != nil {
				t.Fatal(err)
			}
		}

		cache, err := cacheFactory(file)
		if err != nil {
			t.Fatal(err)
		}

		// this will kick of the lifecycle functions of templates.Driver
		// During the initialization, the mutex of PersistentCache denies access to any other (synchronous)
		// functions, such that Get, Add, etc. are blocking until initialize returns.
		//
		// Because we are starting with a cancellable context, we do not call cache.Close(...), but just
		// cancel the context from here, which will terminate the goroutine.
		go cache.Start(context.Background())

		tts := []TemplateRecord{
			{
				TemplateId: 300,
				Fields: []Field{
					NewFieldBuilder(IPFIX()[2]).Length(4).Complete(),
					NewFieldBuilder(IPFIX()[150]).Length(4).Complete(),
					NewFieldBuilder(IPFIX()[10]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[14]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[4]).Length(1).Complete(),
					NewFieldBuilder(IPFIX()[6]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[1]).Length(4).Complete(),
					NewFieldBuilder(IPFIX()[7]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[11]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[8]).Length(4).Complete(),
					NewFieldBuilder(IPFIX()[12]).Length(4).Complete(),
				},
			},
			{
				TemplateId: 301,
				Fields: []Field{
					NewFieldBuilder(IPFIX()[14]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[4]).Length(1).Complete(),
					NewFieldBuilder(IPFIX()[6]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[1]).Length(4).Complete(),
					NewFieldBuilder(IPFIX()[7]).Length(2).Complete(),
				},
			},
		}
		otts := []OptionsTemplateRecord{
			{
				TemplateId:      302,
				FieldCount:      9,
				ScopeFieldCount: 2,
				Scopes: []Field{
					NewFieldBuilder(IPFIX()[346]).Length(4).Complete(),
					NewFieldBuilder(IPFIX()[303]).Length(2).Complete(),
				},
				Options: []Field{
					NewFieldBuilder(IPFIX()[339]).Length(1).Complete(),
					NewFieldBuilder(IPFIX()[344]).Length(1).Complete(),
					NewFieldBuilder(IPFIX()[345]).Length(2).Complete(),
					NewFieldBuilder(IPFIX()[342]).Length(8).Complete(),
					NewFieldBuilder(IPFIX()[343]).Length(8).Complete(),
					NewFieldBuilder(IPFIX()[341]).Length(FieldVariableLength).Complete(),
					NewFieldBuilder(IPFIX()[340]).Length(FieldVariableLength).Complete(),
				},
			},
		}
		for _, tt := range tts {
			// we need to copy the TemplateRecord once, otherwise we add the loop variable as a template, which
			// will mutate during loop execution, subsequently mutating the template in the cache
			// Note that this does not occur when decoding templates, as here, the template is fresh each time
			l := tt
			err := cache.Add(context.TODO(), TemplateKey{
				ObservationDomainId: 0,
				TemplateId:          tt.TemplateId,
			}, &Template{
				Record: &l,
			})
			if err != nil {
				t.Error(err)
			}
		}
		for _, ott := range otts {
			// we need to copy the TemplateRecord once, otherwise we add the loop variable as a template, which
			// will mutate during loop execution, subsequently mutating the template in the cache
			l := ott
			err := cache.Add(context.TODO(), TemplateKey{
				ObservationDomainId: 0,
				TemplateId:          ott.TemplateId,
			}, &Template{
				Record: &l,
			})
			if err != nil {
				t.Error(err)
			}
		}

		_, err = cache.Get(context.Background(), NewKey(0, 302))
		if err != nil {
			t.Error(err)
		}

		// t.Log("GET template with id 302 from cache")

		err = cache.Close(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		// clean up
		err = func() error {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			b, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			if len(b) == 0 {
				return errors.New("expected _templates.json to not be empty")
			}
			t.Log(string(b))
			return os.Remove(p)
		}()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("with restore", func(t *testing.T) {
		// create temporary file for fixtures
		p := path.Join(t.TempDir(), "fixture_persistent_test_with_restore.json")
		t.Log(p)
		err := func() error {
			file, err := os.Create(p)
			if err != nil {
				return err
			}
			_, err = file.Write(fixtureTemplates)
			if err != nil {
				return err
			}
			err = file.Close()
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			t.Fatal(err)
		}

		file, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}

		cache, err := cacheFactory(file)
		if err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		go cache.Start(ctx)

		m := cache.GetAll(ctx)

		if len(m) == 0 {
			t.Fatal("found empty map of templates, expected 3 templates to be restored from fixture")
		}

		err = cache.Close(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// clean up
		err = func() error {
			return os.Remove(p)
		}()
		if err != nil {
			t.Fatal(err)
		}
	})
}

var fixtureTemplates []byte = []byte(`
{
  "exported_at": "2023-05-23T16:19:11.98974279+02:00",
  "store_type": "persistent/in_memory",
  "store_name": "persistence_test/in_memory",
  "templates": {
    "0-300": {
      "kind": "TemplateRecord",
      "record": {
        "template_id": 300,
        "fields": [
          {
            "id": 2,
            "name": "packetDeltaCount",
            "length": 4,
            "type": "unsigned64"
          },
          {
            "id": 150,
            "name": "flowStartSeconds",
            "length": 4,
            "type": "dateTimeSeconds"
          },
          {
            "id": 10,
            "name": "ingressInterface",
            "length": 2,
            "type": "unsigned32"
          },
          {
            "id": 14,
            "name": "egressInterface",
            "length": 2,
            "type": "unsigned32"
          },
          {
            "id": 4,
            "name": "protocolIdentifier",
            "length": 1,
            "type": "unsigned8"
          },
          {
            "id": 6,
            "name": "tcpControlBits",
            "length": 2,
            "type": "unsigned16"
          },
          {
            "id": 1,
            "name": "octetDeltaCount",
            "length": 4,
            "type": "unsigned64"
          },
          {
            "id": 7,
            "name": "sourceTransportPort",
            "length": 2,
            "type": "unsigned16"
          },
          {
            "id": 11,
            "name": "destinationTransportPort",
            "length": 2,
            "type": "unsigned16"
          },
          {
            "id": 8,
            "name": "sourceIPv4Address",
            "length": 4,
            "type": "ipv4Address"
          },
          {
            "id": 12,
            "name": "destinationIPv4Address",
            "length": 4,
            "type": "ipv4Address"
          }
        ]
      }
    },
    "0-301": {
      "kind": "TemplateRecord",
      "record": {
        "template_id": 301,
        "fields": [
          {
            "id": 14,
            "name": "egressInterface",
            "length": 2,
            "type": "unsigned32"
          },
          {
            "id": 4,
            "name": "protocolIdentifier",
            "length": 1,
            "type": "unsigned8"
          },
          {
            "id": 6,
            "name": "tcpControlBits",
            "length": 2,
            "type": "unsigned16"
          },
          {
            "id": 1,
            "name": "octetDeltaCount",
            "length": 4,
            "type": "unsigned64"
          },
          {
            "id": 7,
            "name": "sourceTransportPort",
            "length": 2,
            "type": "unsigned16"
          }
        ]
      }
    },
    "0-302": {
      "kind": "OptionsTemplateRecord",
      "record": {
        "template_id": 302,
        "scopes": [
          {
            "id": 346,
            "name": "privateEnterpriseNumber",
            "length": 4,
            "type": "unsigned32"
          },
          {
            "id": 303,
            "name": "informationElementId",
            "length": 2,
            "type": "unsigned16"
          }
        ],
        "options": [
          {
            "id": 339,
            "name": "informationElementDataType",
            "length": 1,
            "type": "unsigned8"
          },
          {
            "id": 344,
            "name": "informationElementSemantics",
            "length": 1,
            "type": "unsigned8"
          },
          {
            "id": 345,
            "name": "informationElementUnits",
            "length": 2,
            "type": "unsigned16"
          },
          {
            "id": 342,
            "name": "informationElementRangeBegin",
            "length": 8,
            "type": "unsigned64"
          },
          {
            "id": 343,
            "name": "informationElementRangeEnd",
            "length": 8,
            "type": "unsigned64"
          },
          {
            "id": 341,
            "name": "informationElementName",
            "length": 65535,
            "is_variable_length": true,
            "type": "string"
          },
          {
            "id": 340,
            "name": "informationElementDescription",
            "length": 65535,
            "is_variable_length": true,
            "type": "string"
          }
        ]
      }
    }
  }
}
`)
