"""
Utility functions and classes for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, TypeVar


class NotGiven:
    """
    A sentinel object to distinguish 'not given' from `None`.
    
    This is used to distinguish between a parameter that was not provided
    and a parameter that was explicitly set to `None`.
    """
    
    def __bool__(self) -> bool:
        return False
    
    def __repr__(self) -> str:
        return "NOT_GIVEN"


NOT_GIVEN = NotGiven()

T = TypeVar("T")


def is_given(obj: T | NotGiven) -> bool:
    """Check if a value is given (not NOT_GIVEN)."""
    return not isinstance(obj, NotGiven)


def maybe_transform(
    data: T | NotGiven,
    transform: Any,
) -> Any:
    """Transform data if it's given, otherwise return NOT_GIVEN."""
    if isinstance(data, NotGiven):
        return NOT_GIVEN
    return transform(data)


def strip_not_given(obj: dict[str, Any]) -> dict[str, Any]:
    """Remove all NOT_GIVEN values from a dictionary."""
    return {key: value for key, value in obj.items() if not isinstance(value, NotGiven)}
