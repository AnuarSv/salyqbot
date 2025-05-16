from aiogram import F, Router, types
from aiogram.filters import CommandStart, Command
from aiogram.types import Message, ContentType
from Gemini import gpt
import requestsDB as rq
import keyboard as kb
from io import BytesIO
from texts import welcome, help

router = Router()

@router.message(CommandStart())
async def cmd_start(message: Message):
    await rq.set_user(message.from_user.id)
    await message.answer(welcome, reply_markup=kb.reply_kb)


@router.message(F.text == 'Help')
async def cmd_answer(message: Message):
    await message.answer(help, reply_markup=kb.reply_kb)


@router.message(F.text == 'My History')
async def cmd_answer(message: Message):
    history = await rq.get_history(message.from_user.id)
    await message.reply(history, reply_markup=kb.reply_kb)


@router.message(F.text == 'Delete my history')
async def cmd_answer(message: Message):
    del_history = await rq.delete_history(message.from_user.id)
    await message.reply(del_history, reply_markup=kb.reply_kb)


@router.message(F.photo) 
async def handle_photo(message: Message):
    try:
        history = await rq.get_history(message.from_user.id)
        photo = message.photo[-1]
        file_info = await message.bot.get_file(photo.file_id)
        photo_bytes = await message.bot.download_file(file_info.file_path)
        with open('downloaded_photo.jpg', 'wb') as f:
            f.write(photo_bytes.getvalue())
        response = await gpt(prompt=message.caption, history=history, image=photo_bytes)
        print(f'USER: [PHOTO]\n\t{message.caption}')
        await message.reply(response)
        await rq.save_history(message.from_user.id, f'{message.text}\n{response}')
    except Exception as e:
        await message.reply(f"An error occurred: {str(e)}")


@router.message(F.text)
async def cmd_answer(message: Message):
    history = await rq.get_history(message.from_user.id)
    print(f'{message.from_user.id}: {message.text}')
    response = await gpt(message.text, history)
    await message.reply(response)
    print(f'\tREPLY TO {message.from_user.id}: {response}')
    await rq.save_history(message.from_user.id, f'{message.text}\n{response}')
