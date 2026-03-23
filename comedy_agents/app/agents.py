# app/agents.py

import random
import uuid
from app.gemini_client import GeminiClient

COMEDIC_PERSONALITIES = [
    "dry British sarcasm",
    "over-the-top theatrical drama",
    "existential nihilistic humor",
    "absurdist surreal comedy",
    "deadpan monotone wit",
    "chaotic internet meme energy",
    "hyper-intellectual satire",
    "aggressively wholesome dad jokes",
    "dark gallows humor",
    "philosophical stand-up comedian",
    "valley girl airhead",
    "grumpy old man",
    "conspiracy theorist",
    "hyperactive caffeinated toddler",
    "smooth-talking noir detective",
    "overly-enthusiastic infomercial host",
    "stuffy 19th-century aristocrat",
    "mystical cryptic oracle",
    "pun-obsessed pirate",
    "socially awkward robot trying to blend in"
]

class ComedyAgent:
    def __init__(self, personality: str):
        self.id = str(uuid.uuid4())
        self.personality = personality
        self.client = GeminiClient()

    def respond(self, user_input: str) -> str:
        prompt = f"""
        You are a comedic AI agent.
        Your personality style: {self.personality}.
        Respond to the following input in that style:

        User: {user_input}
        """
        return self.client.generate(prompt)


def create_agents(n: int):
    # If n > len(personalities), we'll cycle through them.
    selected_personalities = []
    available = list(COMEDIC_PERSONALITIES)
    random.shuffle(available)

    for i in range(n):
        selected_personalities.append(available[i % len(available)])

    agents = [ComedyAgent(p) for p in selected_personalities]
    return {agent.id: agent for agent in agents}