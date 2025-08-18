#!/usr/bin/env python3
# agent_client.py

import json
import requests
import time
import websocket
import threading
from typing import Dict, List, Optional, Any, Union, Callable


class ConversationMessage:
    """Represents a message in the conversation history."""
    
    def __init__(self, role: str, content: str, timestamp: Optional[int] = None):
        """
        Initialize a conversation message.
        
        Args:
            role: The role of the message sender (user, assistant, system)
            content: The content of the message
            timestamp: The timestamp of the message (defaults to current time)
        """
        self.role = role
        self.content = content
        self.timestamp = timestamp or int(time.time())
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert the message to a dictionary."""
        return {
            "role": self.role,
            "content": self.content,
            "timestamp": self.timestamp
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ConversationMessage':
        """Create a message from a dictionary."""
        return cls(
            role=data["role"],
            content=data["content"],
            timestamp=data["timestamp"]
        )


class ToolCall:
    """Represents a call to a tool during inference."""
    
    def __init__(self, name: str, input_data: Dict[str, Any], output: Any, timestamp: Optional[int] = None):
        """
        Initialize a tool call.
        
        Args:
            name: The name of the tool
            input_data: The input to the tool
            output: The output from the tool
            timestamp: The timestamp of the tool call (defaults to current time)
        """
        self.name = name
        self.input = input_data
        self.output = output
        self.timestamp = timestamp or int(time.time())
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert the tool call to a dictionary."""
        return {
            "name": self.name,
            "input": self.input,
            "output": self.output,
            "timestamp": self.timestamp
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ToolCall':
        """Create a tool call from a dictionary."""
        return cls(
            name=data["name"],
            input_data=data["input"],
            output=data["output"],
            timestamp=data["timestamp"]
        )


class InferenceResponse:
    """Represents a response from the agent."""
    
    def __init__(self, output: str, tool_calls: Optional[List[ToolCall]] = None, 
                 reasoning: Optional[str] = None, metadata: Optional[Dict[str, Any]] = None):
        """
        Initialize an inference response.
        
        Args:
            output: The output text from the agent
            tool_calls: The tool calls made during inference
            reasoning: The reasoning trace (if enabled)
            metadata: Additional metadata about the response
        """
        self.output = output
        self.tool_calls = tool_calls or []
        self.reasoning = reasoning
        self.metadata = metadata or {}
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'InferenceResponse':
        """Create an inference response from a dictionary."""
        tool_calls = None
        if "toolCalls" in data and data["toolCalls"]:
            tool_calls = [ToolCall.from_dict(tc) for tc in data["toolCalls"]]
        
        return cls(
            output=data["output"],
            tool_calls=tool_calls,
            reasoning=data.get("reasoning"),
            metadata=data.get("metadata", {})
        )


class AgentCapabilities:
    """Represents the capabilities of an agent."""
    
    def __init__(self, supports_streaming: bool, supports_tool_calls: bool, 
                 supports_reasoning: bool, max_context_length: int, 
                 supported_parameters: List[str]):
        """
        Initialize agent capabilities.
        
        Args:
            supports_streaming: Whether the agent supports streaming responses
            supports_tool_calls: Whether the agent supports tool calls
            supports_reasoning: Whether the agent supports reasoning traces
            max_context_length: The maximum context length supported by the agent
            supported_parameters: The supported inference parameters
        """
        self.supports_streaming = supports_streaming
        self.supports_tool_calls = supports_tool_calls
        self.supports_reasoning = supports_reasoning
        self.max_context_length = max_context_length
        self.supported_parameters = supported_parameters
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'AgentCapabilities':
        """Create agent capabilities from a dictionary."""
        return cls(
            supports_streaming=data["supportsStreaming"],
            supports_tool_calls=data["supportsToolCalls"],
            supports_reasoning=data["supportsReasoning"],
            max_context_length=data["maxContextLength"],
            supported_parameters=data["supportedParameters"]
        )


class TerminalSession:
    """Represents a terminal session with an agent."""

    def __init__(self, terminal_id: str, session_id: str, rows: int, cols: int, client: 'AgentClient'):
        """
        Initialize a terminal session.

        Args:
            terminal_id: The ID of the terminal session
            session_id: The agent session ID
            rows: The number of rows in the terminal
            cols: The number of columns in the terminal
            client: The agent client instance
        """
        self.terminal_id = terminal_id
        self.session_id = session_id
        self.rows = rows
        self.cols = cols
        self.client = client
        self.ws = None
        self.output_callback = None
        self.is_connected = False

    def write(self, data: str) -> Dict[str, Any]:
        """
        Write data to the terminal.

        Args:
            data: The data to write

        Returns:
            The response from the server
        """
        return self.client.write_to_terminal(self.session_id, self.terminal_id, data)

    def read(self) -> str:
        """
        Read data from the terminal.

        Returns:
            The terminal output
        """
        return self.client.read_from_terminal(self.session_id, self.terminal_id)

    def resize(self, rows: int, cols: int) -> Dict[str, Any]:
        """
        Resize the terminal.

        Args:
            rows: The new number of rows
            cols: The new number of columns

        Returns:
            The response from the server
        """
        self.rows = rows
        self.cols = cols
        return self.client.resize_terminal(self.session_id, self.terminal_id, rows, cols)

    def connect_websocket(self, output_callback: Callable[[str], None] = None):
        """
        Connect to the terminal via WebSocket for real-time communication.

        Args:
            output_callback: Callback function to handle terminal output
        """
        self.output_callback = output_callback
        ws_url = f"{self.client.base_url.replace('http', 'ws')}/v1/terminal/ws?sessionId={self.session_id}&terminalId={self.terminal_id}"

        def on_message(ws, message):
            if self.output_callback:
                self.output_callback(message)

        def on_open(ws):
            self.is_connected = True

        def on_close(ws, close_status_code, close_msg):
            self.is_connected = False

        def on_error(ws, error):
            print(f"WebSocket error: {error}")
            self.is_connected = False

        self.ws = websocket.WebSocketApp(
            ws_url,
            header=[f"Authorization: Bearer {self.client.api_key}"],
            on_message=on_message,
            on_open=on_open,
            on_close=on_close,
            on_error=on_error
        )

        # Start WebSocket in a separate thread
        ws_thread = threading.Thread(target=self.ws.run_forever)
        ws_thread.daemon = True
        ws_thread.start()

    def send_websocket(self, data: str):
        """
        Send data via WebSocket.

        Args:
            data: The data to send
        """
        if self.ws and self.is_connected:
            self.ws.send(data)

    def close(self) -> Dict[str, Any]:
        """
        Close the terminal session.

        Returns:
            The response from the server
        """
        if self.ws:
            self.ws.close()
        return self.client.close_terminal(self.session_id, self.terminal_id)


class AgentClient:
    """Client for interacting with the Agent Inferencer API."""
    
    def __init__(self, base_url: str, api_key: str):
        """
        Initialize the agent client.
        
        Args:
            base_url: The base URL of the Agent Inferencer API
            api_key: The API key for authentication
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json"
        })
    
    def list_agents(self) -> List[str]:
        """
        List available agents.
        
        Returns:
            A list of available agent IDs
        """
        response = self.session.get(f"{self.base_url}/v1/agents")
        response.raise_for_status()
        return response.json()["agents"]
    
    def activate_agent(self, agent_id: str, version: str, session_id: str, 
                       config: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Activate an agent for a session.
        
        Args:
            agent_id: The ID of the agent to activate
            version: The version of the agent to activate
            session_id: The session ID to use
            config: Additional configuration for the agent
        
        Returns:
            The activation response
        """
        data = {
            "agentId": agent_id,
            "version": version,
            "sessionId": session_id
        }
        if config:
            data["config"] = config
        
        response = self.session.post(f"{self.base_url}/v1/agents/activate", json=data)
        response.raise_for_status()
        return response.json()
    
    def deactivate_agent(self, session_id: str) -> Dict[str, Any]:
        """
        Deactivate an agent for a session.
        
        Args:
            session_id: The session ID to deactivate
        
        Returns:
            The deactivation response
        """
        data = {
            "sessionId": session_id
        }
        
        response = self.session.post(f"{self.base_url}/v1/agents/deactivate", json=data)
        response.raise_for_status()
        return response.json()
    
    def process_inference(self, session_id: str, input_text: str, 
                          history: Optional[List[ConversationMessage]] = None,
                          parameters: Optional[Dict[str, Any]] = None) -> InferenceResponse:
        """
        Process an inference request.
        
        Args:
            session_id: The session ID to use
            input_text: The input text for the agent
            history: The conversation history
            parameters: Additional parameters for the inference
        
        Returns:
            The inference response
        """
        data = {
            "sessionId": session_id,
            "input": input_text
        }
        
        if history:
            data["history"] = [msg.to_dict() for msg in history]
        
        if parameters:
            data["parameters"] = parameters
        
        response = self.session.post(f"{self.base_url}/v1/inference", json=data)
        response.raise_for_status()
        return InferenceResponse.from_dict(response.json())
    
    def get_agent_schema(self, session_id: str) -> Dict[str, Any]:
        """
        Get the schema of an agent.
        
        Args:
            session_id: The session ID to use
        
        Returns:
            The agent schema
        """
        response = self.session.get(f"{self.base_url}/v1/schema?sessionId={session_id}")
        response.raise_for_status()
        return response.json()
    
    def get_agent_capabilities(self, session_id: str) -> AgentCapabilities:
        """
        Get the capabilities of an agent.
        
        Args:
            session_id: The session ID to use
        
        Returns:
            The agent capabilities
        """
        response = self.session.get(f"{self.base_url}/v1/capabilities?sessionId={session_id}")
        response.raise_for_status()
        return AgentCapabilities.from_dict(response.json())
    
    def get_memory(self, session_id: str, key: str) -> Any:
        """
        Get a value from the agent's memory.
        
        Args:
            session_id: The session ID to use
            key: The key to get
        
        Returns:
            The memory value
        """
        response = self.session.get(f"{self.base_url}/v1/memory?sessionId={session_id}&key={key}")
        response.raise_for_status()
        return response.json()["value"]
    
    def set_memory(self, session_id: str, key: str, value: Any) -> Dict[str, Any]:
        """
        Set a value in the agent's memory.
        
        Args:
            session_id: The session ID to use
            key: The key to set
            value: The value to set
        
        Returns:
            The response
        """
        data = {
            "value": value
        }
        
        response = self.session.post(f"{self.base_url}/v1/memory?sessionId={session_id}&key={key}", json=data)
        response.raise_for_status()
        return response.json()
    
    def get_tee_info(self, session_id: str) -> Dict[str, Any]:
        """
        Get information about the TEE for an agent.
        
        Args:
            session_id: The session ID to use
        
        Returns:
            The TEE information
        """
        response = self.session.get(f"{self.base_url}/v1/tee?sessionId={session_id}")
        response.raise_for_status()
        return response.json()

    # Terminal management methods

    def create_terminal(self, session_id: str, rows: int = 24, cols: int = 80) -> TerminalSession:
        """
        Create a new terminal session.

        Args:
            session_id: The session ID to use
            rows: The number of rows in the terminal (default: 24)
            cols: The number of columns in the terminal (default: 80)

        Returns:
            A TerminalSession object
        """
        data = {
            "sessionId": session_id,
            "rows": rows,
            "cols": cols
        }

        response = self.session.post(f"{self.base_url}/v1/terminal/create", json=data)
        response.raise_for_status()
        result = response.json()

        return TerminalSession(
            terminal_id=result["terminalId"],
            session_id=session_id,
            rows=rows,
            cols=cols,
            client=self
        )

    def resize_terminal(self, session_id: str, terminal_id: str, rows: int, cols: int) -> Dict[str, Any]:
        """
        Resize a terminal session.

        Args:
            session_id: The session ID to use
            terminal_id: The terminal ID to resize
            rows: The new number of rows
            cols: The new number of columns

        Returns:
            The response from the server
        """
        data = {
            "sessionId": session_id,
            "terminalId": terminal_id,
            "rows": rows,
            "cols": cols
        }

        response = self.session.post(f"{self.base_url}/v1/terminal/resize", json=data)
        response.raise_for_status()
        return response.json()

    def write_to_terminal(self, session_id: str, terminal_id: str, data: str) -> Dict[str, Any]:
        """
        Write data to a terminal session.

        Args:
            session_id: The session ID to use
            terminal_id: The terminal ID to write to
            data: The data to write

        Returns:
            The response from the server
        """
        request_data = {
            "sessionId": session_id,
            "terminalId": terminal_id,
            "data": data
        }

        response = self.session.post(f"{self.base_url}/v1/terminal/write", json=request_data)
        response.raise_for_status()
        return response.json()

    def read_from_terminal(self, session_id: str, terminal_id: str) -> str:
        """
        Read data from a terminal session.

        Args:
            session_id: The session ID to use
            terminal_id: The terminal ID to read from

        Returns:
            The terminal output
        """
        response = self.session.get(f"{self.base_url}/v1/terminal/read?sessionId={session_id}&terminalId={terminal_id}")
        response.raise_for_status()
        return response.json()["data"]

    def close_terminal(self, session_id: str, terminal_id: str) -> Dict[str, Any]:
        """
        Close a terminal session.

        Args:
            session_id: The session ID to use
            terminal_id: The terminal ID to close

        Returns:
            The response from the server
        """
        data = {
            "sessionId": session_id,
            "terminalId": terminal_id
        }

        response = self.session.post(f"{self.base_url}/v1/terminal/close", json=data)
        response.raise_for_status()
        return response.json()


# Example usage
if __name__ == "__main__":
    # Create a client
    client = AgentClient("http://localhost:8080", "test-api-key")
    
    # List available agents
    agents = client.list_agents()
    print(f"Available agents: {agents}")
    
    # Create a session ID
    session_id = f"session-{int(time.time())}"
    
    # Activate an agent
    client.activate_agent("example", "1.0", session_id)
    
    try:
        # Get agent capabilities
        capabilities = client.get_agent_capabilities(session_id)
        print(f"Agent capabilities: {capabilities.__dict__}")
        
        # Process an inference request
        response = client.process_inference(session_id, "Hello, world!")
        print(f"Response: {response.output}")
        
        # Set a memory value
        client.set_memory(session_id, "greeting", "Hello, world!")
        
        # Get a memory value
        greeting = client.get_memory(session_id, "greeting")
        print(f"Greeting: {greeting}")

        # Create a terminal session
        terminal = client.create_terminal(session_id, rows=24, cols=80)
        print(f"Created terminal: {terminal.terminal_id}")

        # Write to the terminal
        terminal.write("echo 'Hello from terminal!'\n")

        # Wait a bit for the command to execute
        time.sleep(0.5)

        # Read from the terminal
        output = terminal.read()
        print(f"Terminal output: {output}")

        # Close the terminal
        terminal.close()
        print("Terminal closed")

    finally:
        # Deactivate the agent
        client.deactivate_agent(session_id)