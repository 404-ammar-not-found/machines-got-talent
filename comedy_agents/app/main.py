# app/main.py

from fastapi import FastAPI, HTTPException
from app.agents import create_agents
from app.schemas import CreateAgentsRequest, ChatRequest

app = FastAPI(title="Comedy AI Agents API")

agents_registry = {}

@app.post("/create_agents")
def create_agents_endpoint(request: CreateAgentsRequest):
    global agents_registry
    agents_registry = create_agents(request.n)
    return {
        "message": f"{len(agents_registry)} agents created.",
        "agents": list(agents_registry.keys())
    }


@app.post("/chat")
def chat_with_agent(request: ChatRequest):
    agent = agents_registry.get(request.agent_id)

    if not agent:
        raise HTTPException(status_code=404, detail="Agent not found")

    response = agent.respond(request.message)

    return {
        "agent": request.agent_id,
        "personality": agent.personality,
        "response": response
    }