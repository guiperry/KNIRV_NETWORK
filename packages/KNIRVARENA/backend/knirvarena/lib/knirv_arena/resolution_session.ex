defmodule KnirvArena.ResolutionSession do
  use GenServer, restart: :temporary
  require Logger

  @socket_path "/var/run/knirv/bridge.sock"
  
  alias KnirvArena.{Validator, AssetExporter, Security.Guard}

  defstruct [
    :error_id,
    :failure_context,
    :bounty_amount,
    :start_time,
    :winning_trajectory,
    players: %{},
    submissions: %{}
  ]

  # Client API
  def start_link(context) do
    GenServer.start_link(__MODULE__, context, name: via_tuple(context["error_id"]))
  end

  defp via_tuple(id), do: {:via, Registry, {KnirvArena.SessionRegistry, id}}

  def validate(error_id, trajectory) do
    GenServer.call(via_tuple(error_id), {:validate_submission, trajectory})
  end

  def add_player(error_id, user_id, reputation_score) do
    GenServer.cast(via_tuple(error_id), {:join_session, user_id, reputation_score})
  end

  @impl true
  def init(context) do
    # Broadcast the new "Error Node" to all public TS clients via Phoenix PubSub
    Phoenix.PubSub.broadcast(KnirvArena.PubSub, "arena:lobby", {:new_bounty, context})
    
    {:ok, %__MODULE__{
      error_id: context["error_id"],
      failure_context: context,
      bounty_amount: context["nrn_fee"],
      start_time: DateTime.utc_now()
    }}
  end

  # Handle player joining the session
  @impl true
  def handle_cast({:join_session, user_id, reputation_score}, state) do
    # Check if player is eligible based on reputation
    case Guard.verify_solver_eligibility(user_id, state.failure_context["bounty_tier"]) do
      :ok ->
        new_players = Map.put(state.players, user_id, %{reputation: reputation_score, joined_at: DateTime.utc_now()})
        {:noreply, %{state | players: new_players}}
      {:error, reason} ->
        Logger.warn("Player #{user_id} failed eligibility check: #{reason}")
        {:noreply, state}
    end
  end

  # Handle solution submission
  @impl true
  def handle_cast({:submit_solution, user_id, trajectory}, state) do
    # Check if player is in the session
    if Map.has_key?(state.players, user_id) do
      # Validate the solution through the Go bridge
      case Validator.request_validation(state.error_id, trajectory) do
        {:ok, merkle_proof} ->
          # Solution is valid, update state and notify
          new_state = %__MODULE__{
            state | 
              winning_trajectory: trajectory,
              submissions: Map.put(state.submissions, user_id, %{trajectory: trajectory, status: :verified, timestamp: DateTime.utc_now()})
          }
          
          # Broadcast resolution achieved
          Phoenix.PubSub.broadcast(KnirvArena.PubSub, "arena:resolution:" <> state.error_id, 
            {:resolution_achieved, %{
              winner: user_id,
              trajectory: trajectory,
              merkle_proof: merkle_proof,
              bounty_amount: state.bounty_amount
            }})
            
          # Export knowledge asset for LoRAX training
          AssetExporter.format_for_lorax(%__MODULE__{
            state | 
              winning_trajectory: trajectory
          })
          
          {:noreply, new_state}
        {:error, reason} ->
          # Solution invalid, record failed attempt
          new_submissions = Map.put(state.submissions, user_id, %{trajectory: trajectory, status: :failed, reason: reason, timestamp: DateTime.utc_now()})
          {:noreply, %{state | submissions: new_submissions}}
      end
    else
      {:noreply, state}  # Player not in session, ignore
    end
  end

  # Handle validation requests via GenServer.call
  @impl true
  def handle_call({:validate_submission, trajectory}, _from, state) do
    # Validate the solution through the Go bridge
    case Validator.request_validation(state.error_id, trajectory) do
      {:ok, merkle_proof} ->
        # Solution is valid
        {:reply, {:ok, :verified}, state}
      {:error, reason} ->
          # Solution invalid
          {:reply, {:error, reason}, state}
    end
  end

  # Handle timeout (simplified - in reality would use :timer.send_interval)
  @impl true
  def handle_info(:check_timeout, state) do
    # Check if session has timed out (e.g., 5 minutes)
    elapsed = DateTime.diff(DateTime.utc_now(), state.start_time, :second)
    if elapsed > 300 do  # 5 minutes timeout
      Logger.info("Resolution session #{state.error_id} timed out")
      {:stop, :timeout, state}
    else
      {:noreply, state}
    end
  end

  # Handle shutdown
  @impl true
  def terminate(_reason, _state) do
    # Cleanup resources if needed
    :ok
  end
end