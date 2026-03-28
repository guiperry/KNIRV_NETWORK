defmodule KnirvArena.BountyListener do
  use GenServer
  require Logger

  @socket_path "/var/run/knirv/arena_in.sock"

  def start_link(_) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  @impl true
  def init(state) do
    # Ensure clean socket state
    if File.regular?(@socket_path) do
      File.rm(@socket_path)
    end

    case :gen_tcp.listen(0, [:local, {:ifaddr, {:local, @socket_path}}, {:active, true}, :binary]) do
      {:ok, listen_socket} ->
        Logger.info("KNIRVARENA: UDS Bridge active at #{@socket_path}")
        {:ok, %{listen_socket: listen_socket}}
      {:error, reason} ->
        Logger.error("Failed to start UDS Bridge: #{inspect(reason)}")
        {:stop, reason}
    end
  end

  @impl true
  def handle_info({:tcp, _port, data}, state) do
    # Decode the FailureContext from the Go binary stream
    with {:ok, context} <- Jason.decode(data),
         {:ok, _session_pid} <- KnirvArena.ResolutionSupervisor.start_session(context) do
      Logger.debug("Bounty Received: #{context["error_id"]} - Spawning ResolutionSession.")
    end
    {:noreply, state}
  end
end