#!/usr/bin/env python3
# llamaindex_integration.py

from typing import Any, Dict, List, Optional, Union, Sequence
from llama_index.llms.base import LLM, ChatMessage, MessageRole, CompletionResponse, ChatResponse, ChatResponseGen, CompletionResponseGen
from llama_index.llms.types import ChatResponseMode, CompletionResponseMode
import time
import uuid

from agent_client import AgentClient, ConversationMessage


class AgentPluginLLM(LLM):
    """LlamaIndex integration for Agent Inferencer."""
    
    def __init__(
        self,
        base_url: str,
        api_key: str,
        agent_id: str,
        version: str = "latest",
        session_id: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None,
        **kwargs
    ):
        """Initialize the AgentPluginLLM.
        
        Args:
            base_url: The base URL of the Agent Inferencer API
            api_key: The API key for authentication
            agent_id: The ID of the agent to use
            version: The version of the agent to use
            session_id: The session ID to use (defaults to a random UUID)
            config: Additional configuration for the agent
        """
        super().__init__(**kwargs)
        self.client = AgentClient(base_url, api_key)
        self.agent_id = agent_id
        self.version = version
        self.session_id = session_id or f"llamaindex-{uuid.uuid4()}"
        self.config = config
        
        # Activate the agent
        self.client.activate_agent(self.agent_id, self.version, self.session_id, self.config)
    
    @property
    def metadata(self) -> Dict[str, Any]:
        """Get the metadata for the LLM."""
        return {
            "agent_id": self.agent_id,
            "version": self.version,
            "session_id": self.session_id,
        }
    
    def _convert_message_role(self, role: MessageRole) -> str:
        """Convert a LlamaIndex message role to a ConversationMessage role.
        
        Args:
            role: The LlamaIndex message role
        
        Returns:
            The ConversationMessage role
        """
        if role == MessageRole.USER:
            return "user"
        elif role == MessageRole.ASSISTANT:
            return "assistant"
        elif role == MessageRole.SYSTEM:
            return "system"
        else:
            return "user"
    
    def _convert_message(self, message: ChatMessage) -> ConversationMessage:
        """Convert a LlamaIndex message to a ConversationMessage.
        
        Args:
            message: The LlamaIndex message
        
        Returns:
            The ConversationMessage
        """
        role = self._convert_message_role(message.role)
        return ConversationMessage(role, message.content)
    
    def _convert_messages(self, messages: Sequence[ChatMessage]) -> List[ConversationMessage]:
        """Convert LlamaIndex messages to ConversationMessages.
        
        Args:
            messages: The LlamaIndex messages
        
        Returns:
            The ConversationMessages
        """
        return [self._convert_message(msg) for msg in messages]
    
    def complete(
        self, prompt: str, formatted: bool = False, **kwargs: Any
    ) -> CompletionResponse:
        """Complete a prompt.
        
        Args:
            prompt: The prompt to complete
            formatted: Whether the prompt is already formatted
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            The completion response
        """
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            prompt,
            parameters=kwargs
        )
        
        return CompletionResponse(text=response.output)
    
    def chat(
        self, messages: Sequence[ChatMessage], **kwargs: Any
    ) -> ChatResponse:
        """Chat with the agent.
        
        Args:
            messages: The messages to send to the agent
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            The chat response
        """
        # Convert the messages to ConversationMessages
        history = self._convert_messages(messages)
        
        # Get the last user message
        last_user_message = None
        for message in reversed(messages):
            if message.role == MessageRole.USER:
                last_user_message = message.content
                break
        
        if last_user_message is None:
            last_user_message = "Hello"
        
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            last_user_message,
            history=history,
            parameters=kwargs
        )
        
        # Create a ChatMessage from the response
        message = ChatMessage(
            role=MessageRole.ASSISTANT,
            content=response.output
        )
        
        return ChatResponse(message=message)
    
    def stream_complete(
        self, prompt: str, formatted: bool = False, **kwargs: Any
    ) -> CompletionResponseGen:
        """Stream a completion.
        
        Args:
            prompt: The prompt to complete
            formatted: Whether the prompt is already formatted
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            A generator of completion responses
        """
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            prompt,
            parameters={**kwargs, "stream": True}
        )
        
        # In a real implementation, we would stream the response
        # For now, we'll just yield the entire response
        yield CompletionResponse(text=response.output)
    
    def stream_chat(
        self, messages: Sequence[ChatMessage], **kwargs: Any
    ) -> ChatResponseGen:
        """Stream a chat response.
        
        Args:
            messages: The messages to send to the agent
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            A generator of chat responses
        """
        # Convert the messages to ConversationMessages
        history = self._convert_messages(messages)
        
        # Get the last user message
        last_user_message = None
        for message in reversed(messages):
            if message.role == MessageRole.USER:
                last_user_message = message.content
                break
        
        if last_user_message is None:
            last_user_message = "Hello"
        
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            last_user_message,
            history=history,
            parameters={**kwargs, "stream": True}
        )
        
        # Create a ChatMessage from the response
        message = ChatMessage(
            role=MessageRole.ASSISTANT,
            content=response.output
        )
        
        # In a real implementation, we would stream the response
        # For now, we'll just yield the entire response
        yield ChatResponse(message=message)
    
    def __del__(self):
        """Clean up resources when the object is deleted."""
        try:
            if hasattr(self, 'client') and hasattr(self, 'session_id'):
                self.client.deactivate_agent(self.session_id)
        except:
            pass


# Example usage
if __name__ == "__main__":
    # Create an LLM
    llm = AgentPluginLLM(
        base_url="http://localhost:8080",
        api_key="test-api-key",
        agent_id="example",
        version="1.0"
    )
    
    # Complete a prompt
    response = llm.complete("Hello, world!")
    print(f"Completion Response: {response.text}")
    
    # Chat with the agent
    messages = [
        ChatMessage(role=MessageRole.SYSTEM, content="You are a helpful assistant."),
        ChatMessage(role=MessageRole.USER, content="Hello, world!")
    ]
    
    response = llm.chat(messages)
    print(f"Chat Response: {response.message.content}")
    
    # Stream a completion
    for response in llm.stream_complete("Hello, world!"):
        print(f"Streaming Completion: {response.text}")
    
    # Stream a chat response
    for response in llm.stream_chat(messages):
        print(f"Streaming Chat: {response.message.content}")