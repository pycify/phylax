1. App calls your backend:
   POST /app-attestation/challenge
2. Backend returns:
   challenge/requestHash
3. Android app calls Play Integrity SDK with that requestHash.
4. Google Play returns an integrityToken.
5. App sends integrityToken to your backend:
   POST /app-attestation/verify
6. Backend calls 
   playintegrity.googleapis.com/v1/{packageName}:decodeIntegrityToken
7. Backend checks:

   - requestPackageName matches your package
   - requestHash matches the challenge
   - appRecognitionVerdict is PLAY_RECOGNIZED
   - deviceRecognitionVerdict includes MEETS_DEVICE_INTEGRITY
   - timestamp is fresh
8. Backend returns app_attestation_token.
