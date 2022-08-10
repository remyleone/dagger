package main

import (
	_ "embed"
	"fmt"
	"go/types"
	"os"
	"path/filepath"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen"
	gqlconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/Khan/genqlient/generate"
	coreschema "github.com/dagger/cloak/api/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

//go:embed go.gotpl
var tmpl string

// TODO: abstract into an interface once support added for more langs (make pluggable, etc.)
func generateGoImplStub() error {
	cfg := gqlconfig.DefaultConfig()
	cfg.Exec = gqlconfig.ExecConfig{Filename: filepath.Join(filepath.Dir(configFile), "_deleteme.go"), Package: "main"}
	cfg.SchemaFilename = gqlconfig.StringList{filepath.Join(filepath.Dir(configFile), "schema.graphql")}
	cfg.Model = gqlconfig.PackageConfig{
		Filename: filepath.Join(generateOutputDir, "models.go"),
		Package:  "main",
	}
	cfg.Models = gqlconfig.TypeMap{
		"SecretID": gqlconfig.TypeMapEntry{
			Model: gqlconfig.StringList{"github.com/dagger/cloak/sdk/go/dagger.SecretID"},
		},
		"FSID": gqlconfig.TypeMapEntry{
			Model: gqlconfig.StringList{"github.com/dagger/cloak/sdk/go/dagger.FSID"},
		},
		"Filesystem": gqlconfig.TypeMapEntry{
			Model: gqlconfig.StringList{"github.com/dagger/cloak/sdk/go/dagger.Filesystem"},
			Fields: map[string]gqlconfig.TypeMapField{
				"exec":        {Resolver: false},
				"dockerbuild": {Resolver: false},
				"file":        {Resolver: false},
			},
		},
		"Exec": gqlconfig.TypeMapEntry{
			Model: gqlconfig.StringList{"github.com/dagger/cloak/sdk/go/dagger.Exec"},
			Fields: map[string]gqlconfig.TypeMapField{
				"fs":       {Resolver: false},
				"stdout":   {Resolver: false},
				"stderr":   {Resolver: false},
				"exitcode": {Resolver: false},
				"mount":    {Resolver: false},
			},
		},
	}

	if err := gqlconfig.CompleteConfig(cfg); err != nil {
		return fmt.Errorf("error completing config: %w", err)
	}
	if err := api.Generate(cfg, api.AddPlugin(plugin{mainPath: filepath.Join(generateOutputDir, "main.go")})); err != nil {
		return fmt.Errorf("error generating code: %w", err)
	}
	_ = os.Remove(cfg.Exec.Filename)
	return nil
}

func generateGoClientStubs(subdir string) error {
	cfg := &generate.Config{
		Schema:     generate.StringList{"schema.graphql"},
		Operations: generate.StringList{"operations.graphql"},
		Generated:  "generated.go",
		Bindings: map[string]*generate.TypeBinding{
			"Filesystem": {Type: "github.com/dagger/cloak/sdk/go/dagger.Filesystem"},
			"Exec":       {Type: "github.com/dagger/cloak/sdk/go/dagger.Exec"},
			"FSID":       {Type: "github.com/dagger/cloak/sdk/go/dagger.FSID"},
			"SecretID":   {Type: "github.com/dagger/cloak/sdk/go/dagger.SecretID"},
		},
		ClientGetter: "github.com/dagger/cloak/sdk/go/dagger.Client",
	}
	if err := cfg.ValidateAndFillDefaults(subdir); err != nil {
		return err
	}
	generated, err := generate.Generate(cfg)
	if err != nil {
		return err
	}
	for filename, content := range generated {
		if err := os.WriteFile(filename, content, 0644); err != nil {
			return err
		}
	}
	return nil
}

type plugin struct {
	mainPath string
}

func (plugin) Name() string {
	return "cloakgen"
}

func (plugin) InjectSourceEarly() *ast.Source {
	// TODO: shouldn't rely on embedded schema from that package in this separate binary
	return &ast.Source{BuiltIn: true, Input: coreschema.Schema}
}

func (p plugin) GenerateCode(data *codegen.Data) error {
	file := File{}

	typesByName := make(map[string]types.Type)
	for _, o := range data.Objects {
		if o.BuiltIn || o.IsReserved() {
			continue
		}
		var hasResolvers bool
		for _, f := range o.Fields {
			if !f.IsReserved() && len(f.Args) > 0 {
				hasResolvers = true
			}
		}
		if !hasResolvers {
			continue
		}
		o.Root = true
		file.Objects = append(file.Objects, o)
		typesByName[o.Reference().String()] = o.Reference()
		for _, f := range o.Fields {
			if len(f.Args) == 0 {
				continue
			}

			f.MethodHasContext = true
			resolver := Resolver{o, f, "", `panic("not implemented")`}
			file.Resolvers = append(file.Resolvers, &resolver)
			for _, arg := range f.Args {
				typesByName[arg.TypeReference.GO.String()] = arg.TypeReference.GO
			}
		}
	}

	resolverBuild := &ResolverBuild{
		File:         &file,
		PackageName:  "main",
		ResolverType: "Resolver",
		HasRoot:      true,
		typesByName:  typesByName,
	}

	return templates.Render(templates.Options{
		PackageName: "main",
		Filename:    p.mainPath,
		Data:        resolverBuild,
		Packages:    data.Config.Packages,
		Template:    tmpl,
	})
}

type ResolverBuild struct {
	*File
	HasRoot      bool
	PackageName  string
	ResolverType string
	typesByName  map[string]types.Type
}

func (r ResolverBuild) ShortTypeName(name string) string {
	shortName := templates.CurrentImports.LookupType(r.typesByName[name])
	if shortName == "*<nil>" {
		shortName = "struct{}"
	}
	return shortName
}

func (r ResolverBuild) PointedToShortTypeName(name string) string {
	t, ok := r.typesByName[name].(*types.Pointer)
	if !ok {
		return ""
	}
	shortName := templates.CurrentImports.LookupType(t.Elem())
	if shortName == "*<nil>" {
		shortName = "struct{}"
	}
	return shortName
}

type File struct {
	// These are separated because the type definition of the resolver object may live in a different file from the
	// resolver method implementations, for example when extending a type in a different graphql schema file
	Objects         []*codegen.Object
	Resolvers       []*Resolver
	RemainingSource string
}

type Resolver struct {
	Object         *codegen.Object
	Field          *codegen.Field
	Comment        string
	Implementation string
}
