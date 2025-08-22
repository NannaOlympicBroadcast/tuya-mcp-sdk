"""
Authentication Manager
"""

import asyncio
import logging
from typing import Optional
# Try to import aiohttp, fallback to placeholder if not available
try:
    import aiohttp
except ImportError:
    # Fallback for when aiohttp is not available
    raise ImportError("aiohttp not available")

from .models import AuthConfig, TokenResponse, TokenData
from .signature import SignatureUtils
from .exceptions import AuthenticationError

logger = logging.getLogger(__name__)


class AuthManager:
    """Authentication Manager"""

    def __init__(self, config: AuthConfig):
        self.config = config
        self._token_data: Optional[TokenData] = None
        self._session: Optional[aiohttp.ClientSession] = None

    async def __aenter__(self):
        self._session = aiohttp.ClientSession()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self._session:
            await self._session.close()

    async def get_token(self, force_refresh: bool = False, max_retries: int = 3) -> TokenData:
        """
        Get access token

        Args:
            force_refresh: Whether to force refresh token
            max_retries: Maximum number of retry attempts

        Returns:
            TokenData object

        Raises:
            AuthenticationError: Raised when authentication fails
        """
        if self._token_data and not force_refresh:
            return self._token_data

        last_error = None

        for attempt in range(max_retries + 1):
            try:
                # Build authentication URL
                auth_url = f"{self.config.endpoint.rstrip('/')}/v1/client/registration"
                if not auth_url.startswith(('http://', 'https://')):
                    # Use schema from config, default to https
                    schema = self.config.schema or "https"
                    auth_url = f"{schema}://{auth_url}"

                # Create signature headers
                headers = SignatureUtils.create_auth_headers(
                    access_id=self.config.access_id,
                    access_secret=self.config.access_secret,
                    uri="/v1/client/registration"
                )

                # Add Accept header for JSON response
                headers['Accept'] = 'application/json'

                logger.info("Getting token (attempt %s/%s): %s",
                            attempt + 1, max_retries + 1, auth_url)

                if not self._session:
                    raise AuthenticationError("HTTP session not initialized")

                async with self._session.get(auth_url, headers=headers) as response:
                    if response.status != 200:
                        error_text = await response.text()
                        last_error = AuthenticationError(
                            f"Failed to get token, status code: {response.status}, response: {error_text}"
                        )

                        # If it's a non-retryable HTTP error, don't retry
                        if response.status in [403, 404]:
                            raise last_error

                        # For 401 errors, we need to check the error_code in response
                        if response.status == 401:
                            try:
                                response_data = await response.json()
                                token_response = TokenResponse(**response_data)
                                error_code = token_response.error_code

                                # If error_code is 10004 with 401 status, don't retry
                                if error_code == "10004":
                                    logger.error(
                                        "Non-retryable error: HTTP 401 with error_code 10004")
                                    raise last_error

                                # For other 401 errors, allow retry
                                if attempt < max_retries:
                                    logger.warning(
                                        "HTTP 401 with error_code %s, retrying... (%s/%s)", error_code, attempt + 1, max_retries)
                                    # Wait 1 second before retry
                                    await asyncio.sleep(1)
                                    continue
                                else:
                                    # Other 401 errors are non-retryable after max retries
                                    raise last_error from None
                            except Exception:
                                # If we can't parse the response, treat as non-retryable 401
                                raise last_error from None

                        # Otherwise, continue to retry
                        if attempt < max_retries:
                            logger.warning(
                                "HTTP %s error, retrying... (%s/%s)", response.status, attempt + 1, max_retries)
                            # Wait 1 second before retry
                            await asyncio.sleep(1)
                            continue
                        raise last_error from None

                    response_data = await response.json()
                    token_response = TokenResponse(**response_data)

                    if not token_response.success:
                        error_msg = token_response.error_msg or token_response.msg or "Unknown error"
                        error_code = token_response.error_code

                        last_error = AuthenticationError(
                            f"Authentication failed: {error_msg} (error_code: {error_code})"
                        )

                        # If error_code is 10004 and we reach here (status 200), don't retry
                        if error_code == "10004":
                            logger.error(
                                "Non-retryable error: HTTP 200 with error_code 10004: %s", error_msg)
                            raise last_error

                        # For other errors, allow retry
                        if attempt < max_retries:
                            logger.warning(
                                "Authentication failed with error_code %s, retrying... (%s/%s)", error_code, attempt + 1, max_retries)
                            # Wait 1 second before retry
                            await asyncio.sleep(1)
                            continue
                        raise last_error

                    # Success case
                    if not token_response.data:
                        raise AuthenticationError(
                            "Token data is missing in successful response")

                    self._token_data = token_response.data
                    logger.info("Token obtained successfully")
                    return self._token_data

            except aiohttp.ClientError as e:
                last_error = AuthenticationError(
                    f"Network request failed: {e}")
                if attempt < max_retries:
                    logger.warning(
                        "Network error, retrying... (%s/%s): %s", attempt + 1, max_retries, e)
                    await asyncio.sleep(1)  # Wait 1 second before retry
                    continue
                raise last_error from None
            except AuthenticationError:
                # Re-raise AuthenticationError as-is
                raise
            except Exception as e:
                last_error = AuthenticationError(
                    f"Error occurred while getting token: {e}")
                if attempt < max_retries:
                    logger.warning(
                        "Unexpected error, retrying... (%s/%s): %s", attempt + 1, max_retries, e)
                    await asyncio.sleep(1)  # Wait 1 second before retry
                    continue
                raise last_error from None

        # This should never be reached, but just in case
        if last_error:
            raise last_error from None
        raise AuthenticationError(
            "Unknown error occurred during token acquisition")

    async def refresh_token(self) -> TokenData:
        """Refresh token"""
        return await self.get_token(force_refresh=True)

    def get_cached_token(self) -> Optional[TokenData]:
        """Get cached token (without network request)"""
        return self._token_data

    def clear_token(self):
        """Clear cached token"""
        self._token_data = None
