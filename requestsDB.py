from models import async_session, User
from sqlalchemy import select


async def set_user(tg_id: int) -> None:
    async with async_session() as session:
        user = await session.scalar(select(User).where(User.tg_id == tg_id))
        if user is None:
            session.add(User(tg_id=tg_id))
            await session.commit()


async def save_history(tg_id: int, text: str) -> None:
    async with async_session() as session:
        user = await session.scalar(select(User).where(User.tg_id == tg_id))
        if user:
            if user.history:
                user.history += f"\n{text}"
            else:
                user.history = f"HISTORY\n{text}"
            await session.commit()


async def get_history(tg_id: int) -> str:
    async with async_session() as session:
        user = await session.scalar(select(User).where(User.tg_id == tg_id))
        return user.history if user and user.history else "None"


async def delete_history(tg_id: int) -> str:
    async with async_session() as session:
        user = await session.scalar(select(User).where(User.tg_id == tg_id))
        if user:
            user.history = ""
            await session.commit()
            return "Done"
        return "User not found"
