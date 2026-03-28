defmodule KnirvArena.Security.Guard do
  @moduledoc """
  Security guard for verifying solver eligibility based on reputation and stake requirements.
  """

  alias KnirvArena.Bridge
  require Logger

  @min_reputation_threshold 50
  @high_value_bounty_threshold 1000  # NRN

  def verify_solver_eligibility(user_id, bounty_tier) do
    # Check reputation score from KNIRVGRAPH (via Go-Bridge)
    case Bridge.get_reputation(user_id) do
      {:ok, score} ->
        cond do
          score < threshold_for(bounty_tier) ->
            {:error, :insufficient_reputation}
          is_blacklisted?(user_id) ->
            {:error, :blacklisted}
          true ->
            :ok
        end
      {:error, reason} ->
        Logger.warn("Could not fetch reputation for user #{user_id}: #{reason}")
        {:error, :reputation_check_failed}
    end
  end

  defp threshold_for(:low) do
    @min_reputation_threshold
  end

  defp threshold_for(:medium) do
    @min_reputation_threshold * 2
  end

  defp threshold_for(:high) do
    @min_reputation_threshold * 4
  end

  defp is_blacklisted?(user_id) do
    # In a real implementation, this would check a blacklist database
    # For now, we'll return false (no one is blacklisted)
    false
  end
end