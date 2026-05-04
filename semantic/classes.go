package semantic

import "jayess-go/ast"

func analyzeClassDeclaration(scope *scope, stmt *ast.ClassDecl) error {
	if err := analyzeOptionalExpression(scope, stmt.SuperClass); err != nil {
		return err
	}
	if err := analyzeConstructableIdentifier(scope, stmt, stmt.SuperClass, "extends target"); err != nil {
		return err
	}
	if stmt.Name != "" && !scope.declareConstructable(stmt.Name) {
		return errorAt(stmt, "duplicate declaration %s", stmt.Name)
	}
	privateMembers, err := collectPrivateMembers(stmt.Members)
	if err != nil {
		return err
	}
	if err := validateConstructors(stmt.Members); err != nil {
		return err
	}
	if err := validateStaticPrototypeMembers(stmt.Members); err != nil {
		return err
	}
	for _, member := range stmt.Members {
		if err := analyzeOptionalExpression(scope, member.KeyExpr); err != nil {
			return err
		}
		if member.StaticBlock {
			context := rootContext().enterClassStaticBlock(stmt.SuperClass != nil, privateMembers)
			if err := analyzeStatements(newScope(scope), context, member.Body); err != nil {
				return err
			}
			continue
		}
		if member.Field {
			context := rootContext().enterClassField(stmt.SuperClass != nil, privateMembers)
			if err := analyzeOptionalExpressionWithContext(scope, context, member.Value); err != nil {
				return err
			}
			continue
		}
		context := rootContext().enterClassMethod(stmt.SuperClass != nil, privateMembers, member.IsAsync, member.IsGenerator, member.Constructor)
		if err := analyzeFunctionBodyWithContext(scope, member.Params, member.Body, context); err != nil {
			return err
		}
	}
	return nil
}

func validateStaticPrototypeMembers(members []ast.ClassMember) error {
	for _, member := range members {
		if member.Static && !member.StaticBlock && member.Name == "prototype" {
			return errorAt(&member, "static class member cannot be named prototype")
		}
	}
	return nil
}

func collectPrivateMembers(members []ast.ClassMember) (map[string]bool, error) {
	privateMembers := map[string]bool{}
	privateKinds := map[string]privateMemberKind{}
	for _, member := range members {
		if !member.Private {
			continue
		}
		kind := privateKind(member)
		if existing, exists := privateKinds[member.Name]; exists && !isAccessorPair(existing, kind) {
			return nil, errorAt(&member, "duplicate private member #%s", member.Name)
		}
		privateKinds[member.Name] |= kind
		privateMembers[member.Name] = true
	}
	return privateMembers, nil
}

type privateMemberKind uint8

const (
	privateField privateMemberKind = 1 << iota
	privateMethod
	privateGetter
	privateSetter
)

func privateKind(member ast.ClassMember) privateMemberKind {
	switch {
	case member.Getter:
		return privateGetter
	case member.Setter:
		return privateSetter
	case member.Field:
		return privateField
	default:
		return privateMethod
	}
}

func isAccessorPair(existing privateMemberKind, next privateMemberKind) bool {
	if existing == privateGetter && next == privateSetter {
		return true
	}
	if existing == privateSetter && next == privateGetter {
		return true
	}
	return false
}

func validateConstructors(members []ast.ClassMember) error {
	seen := false
	for _, member := range members {
		if !member.Constructor {
			continue
		}
		if member.IsAsync || member.IsGenerator {
			return errorAt(&member, "constructor cannot be async or generator")
		}
		if seen {
			return errorAt(&member, "duplicate constructor")
		}
		seen = true
	}
	return nil
}
