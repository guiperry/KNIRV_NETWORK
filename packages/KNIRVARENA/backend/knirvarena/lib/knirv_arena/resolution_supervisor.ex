defmodule KnirvArena.ResolutionSupervisor do
  use DynamicSupervisor

  def start_link(_) do
    DynamicSupervisor.start_link(__MODULE__, :ok, name: __MODULE__)
  end

  def init(:ok) do
    DynamicSupervisor.init(strategy: :one_for_one)
  end

  def start_session(context) do
    # Start the ResolutionSession GenServer under this DynamicSupervisor
    spec = {KnirvArena.ResolutionSession, context}
    DynamicSupervisor.start_child(__MODULE__, spec)
  end
end