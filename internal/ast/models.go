package ast

import (
	"go/ast"

	"github.com/neuronlabs/inflection"
	"golang.org/x/tools/go/packages"

	"github.com/neuronlabs/neuron-generator/input"
)

func (g *ModelGenerator) extractFileModels(d *ast.GenDecl, file *ast.File, pkg *packages.Package) (models []*input.Model) {
	for _, spec := range d.Specs {
		switch st := spec.(type) {
		case *ast.TypeSpec:
			structType, ok := st.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if st.Name == nil {
				continue
			}
			modelName := st.Name.Name
			if len(g.Types) != 0 {
				var matchedModel string
				for i, tp := range g.Types {
					if tp == st.Name.Name {
						matchedModel = tp
						g.Types = append(g.Types[:i], g.Types[i+1:]...)
						break
					}
				}
				if matchedModel == "" {
					continue
				}
			} else if len(g.Exclude) != 0 {
				var matchedModel string
				for _, tp := range g.Exclude {
					if tp == st.Name.Name {
						matchedModel = tp
						break
					}
				}
				if matchedModel != "" {
					continue
				}
			}

			if model := g.extractModel(file, structType, pkg, modelName); model != nil {
				models = append(models, model)
			}
		default:
			continue
		}
	}
	return models
}

func (g *ModelGenerator) extractModel(file *ast.File, structType *ast.StructType, pkg *packages.Package, modelName string) (model *input.Model) {
	model = &input.Model{
		CollectionName: g.namerFunc(inflection.Plural(modelName)),
		Name:           modelName,
		Receivers:      make(map[string]int),
		PackageName:    pkg.Name,
	}

	// Find primary field key.
	for i, structField := range structType.Fields.List {
		if len(structField.Names) == 0 {
			// Embedded fields are not taken into account.
			continue
		}
		name := structField.Names[0]
		if !name.IsExported() {
			// Private fields are not taken into account.
			continue
		}

		if !isExported(structField) {
			continue
		}
		field := input.Field{
			Index:      i,
			Name:       name.String(),
			NeuronName: g.namerFunc(name.String()),
			Type:       fieldTypeName(structField.Type),
			Model:      model,
			Ast:        structField,
		}

		// Set the Tags for given field.
		if structField.Tag != nil {
			field.Tags = structField.Tag.Value
			tags := extractTags(field.Tags, "neuron", ";", ",")
			for _, tag := range tags {
				if tag.key == "-" {
					continue
				}
			}
		}

		if isFieldRelation(structField) {
			field.IsSlice = isMany(structField.Type)
			field.IsElemPointer = isElemPointer(structField)
			field.IsPointer = isPointer(structField)
			model.Relations = append(model.Relations, &field)
			continue
		} else if importedField := g.isImported(file, structField); importedField != nil {
			importedField.Field = &field
			importedField.AstField = structField
			if isPrimary(structField) {
				model.Primary = importedField.Field
			}
			g.modelImportedFields[model] = append(g.modelImportedFields[model], importedField)
			continue
		}
		fieldPtr := &field
		g.setModelField(structField, fieldPtr, false)
		// Check if field is a primary key field.
		if isPrimary(structField) {
			model.Primary = fieldPtr
		}
		model.Fields = append(model.Fields, fieldPtr)
	}

	if model.Primary == nil {
		return nil
	}
	defaultModelPackages := []string{
		"github.com/neuronlabs/neuron/errors",
		"github.com/neuronlabs/neuron/mapping",
	}
	for _, pkg := range defaultModelPackages {
		model.Imports.Add(pkg)
	}

	g.models[model.Name] = model

	for _, relation := range model.Relations {
		if relation.IsSlice {
			model.MultiRelationer = true
		} else {
			model.SingleRelationer = true
		}
	}
	if len(model.Fields) > 0 {
		model.Fielder = true
	}

	for _, importedField := range g.modelImportedFields[model] {
		g.imports[importedField.Path] = importedField.Ident.Name
		pkgTypes := g.importFields[importedField.Path]
		if pkgTypes == nil {
			pkgTypes = map[string][]*ast.Ident{}
		}
		pkgTypes[importedField.Ident.Name] = append(pkgTypes[importedField.Ident.Name], importedField.Ident)
		g.importFields[importedField.Path] = pkgTypes
	}
	return model
}

func getArraySize(expr ast.Expr) string {
	sl, ok := expr.(*ast.ArrayType)
	if !ok {
		return ""
	}
	if bl, ok := sl.Len.(*ast.BasicLit); ok {
		return bl.Value
	}
	return ""

}
