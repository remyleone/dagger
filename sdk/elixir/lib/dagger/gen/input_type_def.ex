# This file generated by `mix dagger.gen`. Please DO NOT EDIT.
defmodule Dagger.InputTypeDef do
  @moduledoc "A graphql input type, which is essentially just a group of named args.\nThis is currently only used to represent pre-existing usage of graphql input types\nin the core API. It is not used by user modules and shouldn't ever be as user\nmodule accept input objects via their id rather than graphql input types."
  use Dagger.QueryBuilder
  @type t() :: %__MODULE__{}
  defstruct [:selection, :client]

  (
    @doc ""
    @spec fields(t()) :: {:ok, [Dagger.FieldTypeDef.t()]} | {:error, term()}
    def fields(%__MODULE__{} = input_type_def) do
      selection = select(input_type_def.selection, "fields")
      selection = select(selection, "description id name typeDef")

      with {:ok, data} <- execute(selection, input_type_def.client) do
        {:ok,
         data
         |> Enum.map(fn value ->
           elem_selection = Dagger.QueryBuilder.Selection.query()
           elem_selection = select(elem_selection, "loadFieldTypeDefFromID")
           elem_selection = arg(elem_selection, "id", value["id"])
           %Dagger.FieldTypeDef{selection: elem_selection, client: input_type_def.client}
         end)}
      end
    end
  )

  (
    @doc "A unique identifier for this InputTypeDef."
    @spec id(t()) :: {:ok, Dagger.InputTypeDefID.t()} | {:error, term()}
    def id(%__MODULE__{} = input_type_def) do
      selection = select(input_type_def.selection, "id")
      execute(selection, input_type_def.client)
    end
  )

  (
    @doc ""
    @spec name(t()) :: {:ok, Dagger.String.t()} | {:error, term()}
    def name(%__MODULE__{} = input_type_def) do
      selection = select(input_type_def.selection, "name")
      execute(selection, input_type_def.client)
    end
  )
end
