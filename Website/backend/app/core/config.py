import os
from pydantic import BaseModel


class Settings(BaseModel):
    env: str = os.getenv("ENV", "production")
    port: int = int(os.getenv("PORT", "8091"))
    database_url: str | None = os.getenv("DATABASE_URL")


settings = Settings()
