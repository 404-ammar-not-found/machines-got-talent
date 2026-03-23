import google.generativeai as genai
import random
import time
from app.config import settings

MOCK_JOKES = [
    'Why did the AI cross the road? To optimize the path to the other side.',
    'I asked the computer for a joke about recursion. It asked me for a joke about recursion.',
    'My girlfriend is like the square root of -100. A solid 10, but also imaginary.',
    'Why dont scientists trust atoms? Because they make up everything!',
    'Parallel lines have so much in common. It is a shame they will never meet.',
    'I am reading a book on anti-gravity. It is impossible to put down!',
    'What is the best thing about Switzerland? I dont know, but the flag is a big plus.'
]

class GeminiClient:
    def __init__(self, model_name='gemini-1.5-flash'):
        self.api_key = settings.GEMINI_API_KEY
        self.mock_mode = True
        if self.api_key:
            try:
                genai.configure(api_key=self.api_key)
                self.model = genai.GenerativeModel(model_name)
                self.mock_mode = False
            except Exception:
                pass

    def generate(self, prompt: str) -> str:
        if self.mock_mode:
            time.sleep(1)
            return random.choice(MOCK_JOKES)
        try:
            response = self.model.generate_content(prompt)
            return response.text
        except Exception:
            return random.choice(MOCK_JOKES)
