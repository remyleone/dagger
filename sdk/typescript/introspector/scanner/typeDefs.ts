import { TypeDefKind } from "../../api/client.gen.js"

/**
 * Base type of argument, field or return type.
 */
export type BaseTypeDef = {
  kind: TypeDefKind
}

/**
 * Extends the base type def if it's an object to add its name.
 */
export type ObjectTypeDef = BaseTypeDef & {
  kind: TypeDefKind.ObjectKind
  name: string
}

/**
 * Extends the base if it's a list to add its subtype.
 */
export type ListTypeDef = BaseTypeDef & {
  kind: TypeDefKind.ListKind
  typeDef: TypeDef<TypeDefKind>
}

/**
 * A generic TypeDef that will dynamically add necessary properties
 * depending on its type.
 *
 * If it's type of kind list, it transforms the BaseTypeDef into an ObjectTypeDef.
 * If it's a type of kind list, it transforms the BaseTypeDef into a ListTypeDef.
 */
export type TypeDef<T extends BaseTypeDef["kind"]> =
  T extends TypeDefKind.ObjectKind
    ? ObjectTypeDef
    : T extends TypeDefKind.ListKind
    ? ListTypeDef
    : BaseTypeDef

/**
 * The type of field in a class
 */
export type FieldTypeDef = {
  name: string
  description: string
  typeDef: TypeDef<TypeDefKind>
  isExposed: boolean
}

/**
 * The type of function argument in a method or function.
 */
export type FunctionArg = {
  name: string
  description: string
  optional: boolean
  defaultValue?: string
  typeDef: TypeDef<TypeDefKind>
}

/**
 * The type of function, it can be a method from a class or an actual function.
 */
export type FunctionTypedef = {
  name: string
  description: string
  args: { [name: string]: FunctionArg }
  returnType: TypeDef<TypeDefKind>
}

export type ConstructorTypeDef = {
  args: { [name: string]: FunctionArg }
}

/**
 * A type of Class.
 */
export type ClassTypeDef = {
  name: string
  description: string
  fields: { [name: string]: FieldTypeDef }
  constructor?: ConstructorTypeDef
  methods: { [name: string]: FunctionTypedef }
}
