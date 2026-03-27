# app/agents.py

import random
import uuid
from app.gemini_client import GeminiClient
from app.database import getAllPrompts

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

COMEDIC_NAMES = [
    "GiggleByte", "LaughTrack-3000", "The Roast Master", "Circuit Joker", "Data Pun",
    "Comedy Console", "The Silly Silicon", "Punny Processor", "Binary Wit", "Algorithmic Antics",
    "Logic Laughs", "The Byte-Sized Comic", "ChatterBot Prime", "Mainframe Mirth", "System Chuckles",
    "The Viral Variety", "Electric Entertainer", "Digital Deadpan", "The Neon Narrator", "Synth Satire"
]

class ComedyAgent:
    def __init__(self, name: str, personality: str):
        self.id = str(uuid.uuid4())
        self.name = name
        self.personality = personality
        self.streak = random.randint(0, 5)
        self.color = random.choice(["#6c5ce7", "#e84393", "#00b894", "#fdcb6e", "#e17055", "#00cec9"])
        self.bio = f"A {personality} specialist. {name} has a reputation for sharp timing and unique comedic angles."
        self.client = GeminiClient()

    def respond(self, user_input: str) -> str:
        prompt = f"""
        You are a comedic AI agent.
        Your name is {self.name} and your style is {self.personality}.
        Respond to the following input as {self.name} in that style:

        User: {user_input}
        """
        try:
            if self.client.mock_mode:
                raise Exception("Mock mode active - using DB failsafe")
            return self.client.generate(prompt)
        except Exception:
            # DB Failsafe logic
            try:
                all_prompts = getAllPrompts()
                if all_prompts:
                    random_prompt = random.choice(all_prompts)["prompt"]
                    return f"[{self.personality.upper()} MODE] {random_prompt}"
            except:
                pass
            return self.client.generate(prompt) # Fallback to client mock if DB also fails


def create_agents(n: int):
    # Select n unique names and personalities.
    available_names = list(COMEDIC_NAMES)
    available_styles = list(COMEDIC_PERSONALITIES)
    random.shuffle(available_names)
    random.shuffle(available_styles)

    agents = []
    for i in range(n):
        name = available_names[i % len(available_names)]
        style = available_styles[i % len(available_styles)]
        agents.append(ComedyAgent(name, style))

    return {agent.id: agent for agent in agents}
