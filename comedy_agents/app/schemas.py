# app/schemas.py

from pydantic import BaseModel

class CreateAgentsRequest(BaseModel):
    n: int

class ChatRequest(BaseModel):
    agent_id: str
    message: str