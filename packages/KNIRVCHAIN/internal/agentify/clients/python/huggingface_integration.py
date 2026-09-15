#!/usr/bin/env python3
# huggingface_integration.py

from typing import Any, Dict, List, Optional, Union, Iterable
import uuid
import time

from agent_client import AgentClient, ConversationMessage


class AgentPluginPipeline:
    """Hugging Face integration for Agent Inferencer."""
    
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
        """Initialize the AgentPluginPipeline.
        
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
        self.session_id = session_id or f"huggingface-{uuid.uuid4()}"
        self.config = config
        
        # Activate the agent
        self.client.activate_agent(self.agent_id, self.version, self.session_id, self.config)
    
    def __call__(
        self,
        inputs: Union[str, List[str]],
        **kwargs
    ) -> Union[Dict[str, Any], List[Dict[str, Any]]]:
        """Call the pipeline.
        
        Args:
            inputs: The input text or list of input texts
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            The pipeline output
        """
        if isinstance(inputs, list):
            return [self._process_single_input(input_text, **kwargs) for input_text in inputs]
        else:
            return self._process_single_input(inputs, **kwargs)
    
    def _process_single_input(self, input_text: str, **kwargs) -> Dict[str, Any]:
        """Process a single input.
        
        Args:
            input_text: The input text
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            The pipeline output
        """
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            input_text,
            parameters=kwargs
        )
        
        # Return the output in a format similar to Hugging Face pipelines
        return {
            "generated_text": response.output,
            "metadata": response.metadata
        }
    
    def __del__(self):
        """Clean up resources when the object is deleted."""
        try:
            if hasattr(self, 'client') and hasattr(self, 'session_id'):
                self.client.deactivate_agent(self.session_id)
        except:
            pass


class AgentPluginConversational:
    """Hugging Face conversational integration for Agent Inferencer."""
    
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
        """Initialize the AgentPluginConversational.
        
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
        self.session_id = session_id or f"huggingface-conv-{uuid.uuid4()}"
        self.config = config
        self.history = []
        
        # Activate the agent
        self.client.activate_agent(self.agent_id, self.version, self.session_id, self.config)
    
    def __call__(
        self,
        inputs: Union[str, Dict[str, Any]],
        **kwargs
    ) -> Dict[str, Any]:
        """Call the conversational pipeline.
        
        Args:
            inputs: The input text or a dictionary with 'text' and 'past_user_inputs' and 'generated_responses'
            **kwargs: Additional parameters to pass to the agent
        
        Returns:
            The conversational output
        """
        if isinstance(inputs, dict):
            # Extract the input text and history from the dictionary
            input_text = inputs.get("text", "")
            past_user_inputs = inputs.get("past_user_inputs", [])
            generated_responses = inputs.get("generated_responses", [])
            
            # Build the history
            history = []
            for user_input, response in zip(past_user_inputs, generated_responses):
                history.append(ConversationMessage("user", user_input))
                history.append(ConversationMessage("assistant", response))
        else:
            # Use the input text directly
            input_text = inputs
            history = self.history
        
        # Process the inference request
        response = self.client.process_inference(
            self.session_id,
            input_text,
            history=history,
            parameters=kwargs
        )
        
        # Update the history
        self.history.append(ConversationMessage("user", input_text))
        self.history.append(ConversationMessage("assistant", response.output))
        
        # Extract past user inputs and generated responses
        past_user_inputs = []
        generated_responses = []
        
        for msg in self.history:
            if msg.role == "user":
                past_user_inputs.append(msg.content)
            elif msg.role == "assistant":
                generated_responses.append(msg.content)
        
        # Return the output in a format similar to Hugging Face conversational pipelines
        return {
            "generated_text": response.output,
            "conversation": {
                "past_user_inputs": past_user_inputs,
                "generated_responses": generated_responses
            }
        }
    
    def __del__(self):
        """Clean up resources when the object is deleted."""
        try:
            if hasattr(self, 'client') and hasattr(self, 'session_id'):
                self.client.deactivate_agent(self.session_id)
        except:
            pass


# Example usage
if __name__ == "__main__":
    # Create a pipeline
    pipeline = AgentPluginPipeline(
        base_url="http://localhost:8080",
        api_key="test-api-key",
        agent_id="example",
        version="1.0"
    )
    
    # Generate text
    result = pipeline("Hello, world!")
    print(f"Pipeline Result: {result['generated_text']}")
    
    # Create a conversational pipeline
    conversational = AgentPluginConversational(
        base_url="http://localhost:8080",
        api_key="test-api-key",
        agent_id="example",
        version="1.0"
    )
    
    # Generate a response
    result = conversational("Hello, world!")
    print(f"Conversational Result: {result['generated_text']}")
    
    # Continue the conversation
    result = conversational("How are you?")
    print(f"Conversational Result: {result['generated_text']}")
    
    # Use the conversation history
    conversation = {
        "text": "What's the weather like?",
        "past_user_inputs": ["Hello", "How are you?"],
        "generated_responses": ["Hi there!", "I'm doing well, thank you for asking!"]
    }
    
    result = conversational(conversation)
    print(f"Conversational Result with History: {result['generated_text']}")