# app/agents.py

import random
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
    "philosophical stand-up comedian"
]

class ComedyAgent:
    def __init__(self, personality: str):
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
    personalities = random.sample(COMEDIC_PERSONALITIES, min(n, len(COMEDIC_PERSONALITIES)))
    return {f"agent_{i}": ComedyAgent(personality)
            for i, personality in enumerate(personalities)}