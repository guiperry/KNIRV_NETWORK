#!/usr/bin/env python3
# langchain_integration.py

from typing import Any, Dict, List, Optional, Union
from langchain.llms.base import LLM
from langchain.callbacks.manager import CallbackManagerForLLMRun
from langchain.schema import AIMessage, HumanMessage, SystemMessage
from langchain.schema.messages import BaseMessage
from langchain.tools import BaseTool
import time
import uuid

from agent_client import AgentClient, ConversationMessage


class AgentPluginLLM(LLM):
    """LangChain integration for Agent Inferencer."""
    
    client: AgentClient
    agent_id: str
    version: str
    session_id: Optional[str] = None
    config: Optional[Dict[str, Any]] = None
    
    @property
    def _llm_type(self) -> str:
        """Return the type of LLM."""
        return "agent_plugin"
    
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
        self.session_id = session_id or f"langchain-{uuid.uuid4()}"
        self.config = config
        
        # Activate the agent
        self.client.activate_agent(self.agent_id, self.version, self.session_id, self.config)
    
    def _call(
        self,
        prompt: str,
        stop: Optional[List[str]] = None,
        run_manager: Optional[CallbackManagerForLLMRun] = None,
        **kwargs
    ) -> str:
        """Call the Agent Inferencer API.
        
        Args:
            prompt: The prompt to send to the agent
            stop: A list of strings to stop generation when encountered
            run_manager: The callback manager for the run
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            The generated text
        """
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            prompt,
            parameters=kwargs
        )
        
        return response.output
    
    def __del__(self):
        """Clean up resources when the object is deleted."""
        try:
            if hasattr(self, 'client') and hasattr(self, 'session_id'):
                self.client.deactivate_agent(self.session_id)
        except:
            pass


class AgentPluginChat:
    """Chat integration for Agent Inferencer."""
    
    def __init__(
        self,
        base_url: str,
        api_key: str,
        agent_id: str,
        version: str = "latest",
        session_id: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None
    ):
        """Initialize the AgentPluginChat.
        
        Args:
            base_url: The base URL of the Agent Inferencer API
            api_key: The API key for authentication
            agent_id: The ID of the agent to use
            version: The version of the agent to use
            session_id: The session ID to use (defaults to a random UUID)
            config: Additional configuration for the agent
        """
        self.client = AgentClient(base_url, api_key)
        self.agent_id = agent_id
        self.version = version
        self.session_id = session_id or f"langchain-chat-{uuid.uuid4()}"
        self.config = config
        self.history = []
        
        # Activate the agent
        self.client.activate_agent(self.agent_id, self.version, self.session_id, self.config)
    
    def add_message(self, message: BaseMessage):
        """Add a message to the chat history.
        
        Args:
            message: The message to add
        """
        self.history.append(message)
    
    def _convert_message(self, message: BaseMessage) -> ConversationMessage:
        """Convert a LangChain message to a ConversationMessage.
        
        Args:
            message: The LangChain message
        
        Returns:
            The ConversationMessage
        """
        if isinstance(message, HumanMessage):
            role = "user"
        elif isinstance(message, AIMessage):
            role = "assistant"
        elif isinstance(message, SystemMessage):
            role = "system"
        else:
            role = "user"
        
        return ConversationMessage(role, message.content)
    
    def _convert_messages(self, messages: List[BaseMessage]) -> List[ConversationMessage]:
        """Convert LangChain messages to ConversationMessages.
        
        Args:
            messages: The LangChain messages
        
        Returns:
            The ConversationMessages
        """
        return [self._convert_message(msg) for msg in messages]
    
    def invoke(self, input_text: str, **kwargs) -> AIMessage:
        """Generate a response to the input text.
        
        Args:
            input_text: The input text
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            The generated response as an AIMessage
        """
        # Add the input message to the history
        self.add_message(HumanMessage(content=input_text))
        
        # Convert the history to ConversationMessages
        history = self._convert_messages(self.history)
        
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            input_text,
            history=history,
            parameters=kwargs
        )
        
        # Create an AIMessage from the response
        ai_message = AIMessage(content=response.output)
        
        # Add the response to the history
        self.add_message(ai_message)
        
        return ai_message
    
    def __del__(self):
        """Clean up resources when the object is deleted."""
        try:
            if hasattr(self, 'client') and hasattr(self, 'session_id'):
                self.client.deactivate_agent(self.session_id)
        except:
            pass


class AgentPluginTool(BaseTool):
    """Tool integration for Agent Inferencer."""
    
    name: str
    description: str
    client: AgentClient
    session_id: str
    tool_name: str
    
    def __init__(
        self,
        name: str,
        description: str,
        client: AgentClient,
        session_id: str,
        tool_name: str
    ):
        """Initialize the AgentPluginTool.
        
        Args:
            name: The name of the tool
            description: The description of the tool
            client: The AgentClient to use
            session_id: The session ID to use
            tool_name: The name of the tool in the agent
        """
        super().__init__(name=name, description=description)
        self.client = client
        self.session_id = session_id
        self.tool_name = tool_name
    
    def _run(self, input_text: str, **kwargs) -> str:
        """Run the tool.
        
        Args:
            input_text: The input text
            **kwargs: Additional parameters to pass to the tool
        
        Returns:
            The tool output
        """
        # Set the tool input in memory
        self.client.set_memory(
            self.session_id,
            f"tool_input:{self.tool_name}",
            {"text": input_text, **kwargs}
        )
        
        # Process the inference request to execute the tool
        response = self.client.process_inference(
            self.session_id,
            f"Execute the {self.tool_name} tool with the input: {input_text}",
            parameters={"execute_tool": self.tool_name}
        )
        
        # Get the tool output from memory
        tool_output = self.client.get_memory(self.session_id, f"tool_output:{self.tool_name}")
        
        # Return the tool output as a string
        if isinstance(tool_output, dict) and "result" in tool_output:
            return str(tool_output["result"])
        
        return str(tool_output)


# Example usage
if __name__ == "__main__":
    # Create an LLM
    llm = AgentPluginLLM(
        base_url="http://localhost:8080",
        api_key="test-api-key",
        agent_id="example",
        version="1.0"
    )
    
    # Generate text
    response = llm("Hello, world!")
    print(f"LLM Response: {response}")
    
    # Create a chat
    chat = AgentPluginChat(
        base_url="http://localhost:8080",
        api_key="test-api-key",
        agent_id="example",
        version="1.0"
    )
    
    # Add a system message
    chat.add_message(SystemMessage(content="You are a helpful assistant."))
    
    # Generate a response
    response = chat.invoke("Hello, world!")
    print(f"Chat Response: {response.content}")
    
    # Create a client for tools
    client = AgentClient("http://localhost:8080", "test-api-key")
    session_id = f"tool-{uuid.uuid4()}"
    client.activate_agent("example", "1.0", session_id)
    
    try:
        # Create a tool
        tool = AgentPluginTool(
            name="search",
            description="Search for information",
            client=client,
            session_id=session_id,
            tool_name="search"
        )
        
        # Use the tool
        result = tool.run("What is the capital of France?")
        print(f"Tool Result: {result}")
    finally:
        # Deactivate the agent
        client.deactivate_agent(session_id)