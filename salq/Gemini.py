import base64
import requests
from config import GEMINI_API
from texts import bot_purpose
from io import BytesIO

async def gpt(prompt: str = '', history: str = '', image: BytesIO = None) -> str:
    try:
        url = f'https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key={GEMINI_API}'

        headers = {
            'Content-Type': 'application/json'
        }

        parts = []

        if image:
            image.seek(0)
            image_base64 = base64.b64encode(image.read()).decode('utf-8')
            parts.append({
                "inline_data": {
                    "mime_type": "image/jpeg",
                    "data": image_base64
                }
            })

        if prompt:
            parts.append({
                "text": f"System: {bot_purpose}\nUser History: {history}\nUser Question: {prompt}"
            })

        payload = {
            "contents": [
                {
                    "parts": parts
                }
            ]
        }

        response = requests.post(url, headers=headers, json=payload)

        if response.status_code == 200:
            return response.json()['candidates'][0]['content']['parts'][0]['text']
        else:
            return f"ERROR {response.status_code}: {response.text}"

    except Exception as e:
        return f"ERROR: {str(e)}"
