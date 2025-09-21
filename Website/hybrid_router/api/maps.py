"""Google Maps API proxy to prevent API key exposure"""
from .config import get_google_maps_api_key

router = APIRouter(prefix="/maps", tags=["maps"])
logger = logging.getLogger(__name__)

# Get Google Maps API key from environment variable
GOOGLE_MAPS_API_KEY = get_google_maps_api_key()


@router.get("/api.js")
async def get_google_maps_api():
    """Proxy Google Maps JavaScript API to prevent API key exposure"""
    if not GOOGLE_MAPS_API_KEY:
        logger.error("GOOGLE_MAPS_API_KEY environment variable not set")
        raise HTTPException(status_code=500, detail="Google Maps API key not configured")

    try:
        async with httpx.AsyncClient() as client:
            # Fetch the Google Maps API JavaScript with our API key
            url = f"https://maps.googleapis.com/maps/api/js?key={GOOGLE_MAPS_API_KEY}&libraries=geometry"
            response = await client.get(url)

            if response.status_code != 200:
                logger.error(f"Google Maps API request failed: {response.status_code}")
                raise HTTPException(status_code=502, detail="Failed to fetch Google Maps API")

            # Return the JavaScript with appropriate headers
            return Response(
                content=response.content,
                media_type="application/javascript",
                headers={
                    "Cache-Control": "public, max-age=3600",  # Cache for 1 hour
                    "Access-Control-Allow-Origin": "*",  # Allow cross-origin for the script
                }
            )

    except httpx.RequestError as e:
        logger.error(f"Request error fetching Google Maps API: {e}")
        raise HTTPException(status_code=502, detail="Failed to fetch Google Maps API")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        raise HTTPException(status_code=500, detail="Internal server error")
