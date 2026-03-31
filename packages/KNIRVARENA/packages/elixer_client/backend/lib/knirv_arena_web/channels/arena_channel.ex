defmodule KnirvArenaWeb.ArenaChannel do
  use KnirvArenaWeb, :channel
  require Logger
  alias KnirvArena.{ResolutionSession}

  def join("arena:resolution:" <> error_id, _payload, socket) do
    # Track player presence for reputation and NRN distribution
    send(self(), :after_join)
    {:ok, assign(socket, :error_id, error_id)}
  end

  def handle_in("submit_solution", %{"trajectory" => trajectory, "user_id" => user_id}, socket) do
    # Route submission to the specific ResolutionSession GenServer for validation
    error_id = socket.assigns.error_id
    
    case ResolutionSession.validate(error_id, trajectory) do
      {:ok, :verified} ->
        broadcast!(socket, "resolution_achieved", %{winner: user_id})
        {:reply, :ok, socket}
      {:error, reason} ->
        {:reply, {:error, %{reason: reason}}, socket}
    end
  end

  def handle_in("join_session", %{"user_id" => user_id, "reputation_score" => reputation_score}, socket) do
    error_id = socket.assigns.error_id
    
    # Add player to the session
    ResolutionSession.add_player(error_id, user_id, reputation_score)
    
    {:reply, :ok, socket}
  end
end