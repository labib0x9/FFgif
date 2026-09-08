# FFGif REST API Specification

> **Base URL**: `http://localhost:8080` (or configured `http://ADDR:PORT`)  
> **API Version**: `v1`  
> **Content-Type**: `application/json`

---

## Overview & Conventions

### Authentication
Protected endpoints require a JWT Bearer token passed in the `Authorization` request header:
```http
Authorization: Bearer <jwt_token>
```

### Standard Error Response Format
All error responses return a standardized JSON envelope:
```json
{
  "error_code": "ERROR_CODE_STRING",
  "message": "Human readable error description",
  "status": 400
}
```

Common status codes:
- `200 OK` - Request succeeded.
- `201 Created` - Resource created successfully.
- `202 Accepted` - Background job accepted for asynchronous processing.
- `204 No Content` - Operation succeeded with no response body.
- `400 Bad Request` - Malformed JSON or missing required fields.
- `401 Unauthorized` - Missing, invalid, or expired JWT token.
- `403 Forbidden` - Insufficient permissions / ownership mismatch.
- `404 Not Found` - Resource does not exist.
- `412 Precondition Failed` - Missing or mismatched `If-Match` ETag header (optimistic locking).
- `422 Unprocessable Entity` - Validation rules failed (e.g. invalid string length, out of range values).
- `429 Too Many Requests` - Rate limit exceeded.
- `500 Internal Server Error` - Server error.

---

## Table of Contents
1. [Authentication (`/auth`)](#1-authentication-auth)
2. [User Profile & Quota (`/users`)](#2-user-profile--quota-users)
3. [Uploads (`/uploads`)](#3-uploads-uploads)
4. [Conversion Jobs (`/jobs`)](#4-conversion-jobs-jobs)
5. [GIF Management (`/gifs`)](#5-gif-management-gifs)
6. [Sharing & Access Control (`/gifs/.../shares` & `/s`)](#6-sharing--access-control-gifs-and-s)

---

## 1. Authentication (`/auth`)

### `POST /auth/signup`
Register a new user account and trigger a verification email.

- **Auth Required**: No
- **Max Body**: 1 MB

#### Request Body
```json
{
  "email": "user@example.com",
  "username": "johndoe",
  "fullname": "John Doe",
  "password": "Password123!"
}
```

#### Responses
- **`201 Created`**
  ```json
  {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "email": "user@example.com",
    "username": "johndoe",
    "fullname": "John Doe"
  }
  ```
- **`400 Bad Request`** / **`422 Unprocessable Entity`** - Email already exists or validation failed.

#### Example Call
```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","username":"johndoe","fullname":"John Doe","password":"Password123!"}'
```

---

### `POST /auth/login`
Authenticate user with email and password to receive a JWT session token.

- **Auth Required**: No
- **Max Body**: 1 MB

#### Request Body
```json
{
  "email": "user@example.com",
  "password": "Password123!"
}
```

#### Responses
- **`200 OK`**
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
  ```
- **`401 Unauthorized`** - Invalid credentials or user email is unverified.

#### Example Call
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"Password123!"}'
```

---

### `POST /auth/logout`
Revoke the current JWT token and add it to the Redis blocklist until expiration.

- **Auth Required**: **Yes** (`Bearer <token>`)

#### Responses
- **`200 OK`**
  ```json
  {
    "message": "logged out"
  }
  ```

#### Example Call
```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer <jwt_token>"
```

---

### `GET /auth/verify`
Verify account email using the token received in verification email.

- **Auth Required**: No
- **Query Parameters**:
  - `token` (string, required): Raw verification token.

#### Responses
- **`200 OK`**
  ```json
  {
    "message": "email verified successfully"
  }
  ```
- **`400 Bad Request`** / **`401 Unauthorized`** - Invalid or expired token.

#### Example Call
```bash
curl -X GET "http://localhost:8080/auth/verify?token=34a9af05-a9b2-4a45-b438-5f10a5b9e380"
```

---

### `POST /auth/verify/resend`
Resend verification token to user email.

- **Auth Required**: No

#### Request Body
```json
{
  "email": "user@example.com"
}
```

#### Responses
- **`200 OK`**
  ```json
  {
    "message": "verification email sent"
  }
  ```

---

### `POST /auth/forgot-password`
Request a password reset email.

- **Auth Required**: No

#### Request Body
```json
{
  "email": "user@example.com"
}
```

#### Responses
- **`200 OK`**
  ```json
  {
    "message": "password reset email sent"
  }
  ```

---

### `GET /auth/reset`
Verify reset password token validity.

- **Auth Required**: No
- **Query Parameters**:
  - `token` (string, required): Password reset token.

#### Responses
- **`200 OK`**
  ```json
  {
    "token": "reset-token-string"
  }
  ```

---

### `POST /auth/reset`
Reset password using reset token.

- **Auth Required**: No

#### Request Body
```json
{
  "token": "reset-token-string",
  "password": "NewSecurePassword123!",
  "confirm_password": "NewSecurePassword123!"
}
```

#### Responses
- **`200 OK`**
  ```json
  {
    "message": "password reset successfully"
  }
  ```

---

## 2. User Profile & Quota (`/users`)

### `GET /users/profile/me`
Retrieve authenticated user's profile information.

- **Auth Required**: **Yes**

#### Responses
- **`200 OK`**
  ```json
  {
    "avatar_url": "https://avatar.com/pic.png",
    "username": "johndoe",
    "fullname": "John Doe",
    "email": "user@example.com",
    "verified": true,
    "updated_at": "2026-09-08T12:00:00.000000000Z"
  }
  ```

#### Example Call
```bash
curl -X GET http://localhost:8080/users/profile/me \
  -H "Authorization: Bearer <jwt_token>"
```

---

### `PATCH /users/profile/me`
Update profile fields with optimistic locking using `If-Match`.

- **Auth Required**: **Yes**
- **Headers**:
  - `If-Match` (string, required): Current `updated_at` RFC3339 timestamp of the profile.

#### Request Body
```json
{
  "fullname": "John Updated Doe",
  "avatar_url": "https://avatar.com/new_pic.png"
}
```

#### Responses
- **`200 OK`**
  ```json
  {
    "avatar_url": "https://avatar.com/new_pic.png",
    "username": "johndoe",
    "fullname": "John Updated Doe",
    "email": "user@example.com",
    "verified": true,
    "updated_at": "2026-09-08T12:05:00.000000000Z"
  }
  ```
- **`412 Precondition Failed`** - Missing or mismatched `If-Match` header.

---

### `GET /users/me/quota`
Retrieve account quota limits and current resource usage.

- **Auth Required**: **Yes**

#### Responses
- **`200 OK`**
  ```json
  {
    "id": 1,
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "used_bytes": 10485760,
    "total_bytes": 104857600,
    "gif_count": 3,
    "gif_limit": 20
  }
  ```

---

### `PATCH /users/change-password`
Change account password.

- **Auth Required**: **Yes**

#### Request Body
```json
{
  "current_password": "OldPassword123!",
  "password": "NewPassword456!",
  "confirm_password": "NewPassword456!"
}
```

#### Responses
- **`200 OK`**
  ```json
  {
    "message": "password changed successfully"
  }
  ```

---

### `DELETE /users/me`
Delete user account permanently.

- **Auth Required**: **Yes**

#### Request Body
```json
{
  "password": "CurrentPassword123!"
}
```

#### Responses
- **`200 OK`** (or `204 No Content`)

---

## 3. Uploads (`/uploads`)

### `POST /uploads`
Generate a presigned PUT URL for direct client-to-MinIO video upload. The backend never touches raw video bytes.

- **Auth Required**: **Yes**

#### Request Body
```json
{
  "filename": "my_video.mp4"
}
```

#### Responses
- **`201 Created`**
  ```json
  {
    "upload_url": "http://127.0.0.1:9000/ffgif-temp/user-id%3Auuid.mp4?X-Amz-Algorithm=...",
    "key": "user-123:video-uuid.mp4",
    "expires_in": 300
  }
  ```

#### Flow
1. Call `POST /uploads` to get `upload_url` and `key`.
2. Upload video binary directly to `upload_url` via HTTP `PUT`.
3. MinIO emits bucket notification to RabbitMQ, triggering preprocessing workers.

---

### `GET /uploads/{key}/status`
Poll video preprocessing / ingestion status.

- **Auth Required**: **Yes**
- **Path Parameters**:
  - `key` (string, required): File key returned from `POST /uploads`.

#### Responses
- **`200 OK`**
  ```json
  {
    "file_key": "user-123:video-uuid.mp4",
    "status": "ok"
  }
  ```
  *(Status values: `uploading`, `processing`, `ok`, `failed`)*

---

### `GET /uploads/{key}/stream`
Get presigned streaming URL for uploaded raw video preview.

- **Auth Required**: **Yes**
- **Path Parameters**:
  - `key` (string, required): File key.

#### Responses
- **`200 OK`**
  ```json
  {
    "url": "http://127.0.0.1:9000/ffgif-temp/user-id%3Avideo.mp4?...",
    "expires_in": 300
  }
  ```

---

### `GET /uploads/last`
Fetch metadata of the last uploaded and preprocessed video for the user.

- **Auth Required**: **Yes**

#### Responses
- **`200 OK`**
  ```json
  {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "key": "user-123:video.mp4",
    "filename": "my_video.mp4",
    "content_type": "video/mp4",
    "size_bytes": 15420100,
    "duration_sec": 12.5,
    "uploaded_at": "2026-09-08T12:00:00Z",
    "thumbnail_url": "https://storage/thumb.jpg"
  }
  ```

---

## 4. Conversion Jobs (`/jobs`)

### `POST /jobs`
Enqueue an asynchronous video-to-GIF conversion job.

- **Auth Required**: **Yes**
- **Headers Returned**:
  - `Location: /jobs/{jobId}/status`

#### Request Body
```json
{
  "upload_key": "user-123:video.mp4",
  "start_time": 1.5,
  "end_time": 6.5,
  "width": 480,
  "fps": 15,
  "loop": true
}
```

| Field | Type | Rules | Description |
|---|---|---|---|
| `upload_key` | string | required | Valid uploaded video file key |
| `start_time` | float | `gte=0` | Start time in seconds |
| `end_time` | float | `gt=0` (`end > start`) | End time in seconds |
| `width` | int | `100 <= width <= 1920` | GIF output width in pixels (aspect ratio preserved) |
| `fps` | int | `1 <= fps <= 30` | Frames per second |
| `loop` | bool | optional | Loop animation infinitely |

#### Responses
- **`202 Accepted`**
  ```json
  {
    "job_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "status": "queued"
  }
  ```

#### Example Call
```bash
curl -X POST http://localhost:8080/jobs \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"upload_key":"user-123:video.mp4","start_time":0,"end_time":5,"width":480,"fps":15,"loop":true}'
```

---

### `GET /jobs/{jobId}/status`
Poll the status of an active or completed conversion job.

- **Auth Required**: **Yes**
- **Path Parameters**:
  - `jobId` (string, required): Conversion Job ID.

#### Responses
- **`200 OK`**
  ```json
  {
    "job_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "status": "completed",
    "gif_id": "user-123:converted_gif.gif",
    "progress": 100,
    "updated_at": "2026-09-08T12:00:10Z"
  }
  ```
  *(Status values: `queued`, `converting`, `completed`, `failed`)*

---

## 5. GIF Management (`/gifs`)

### `GET /gifs/me`
List all GIFs owned by the authenticated user with optional visibility filter.

- **Auth Required**: **Yes**
- **Query Parameters**:
  - `filter` (string, optional): `all` (default), `public`, or `private`.

#### Responses
- **`200 OK`**
  ```json
  {
    "data": [
      {
        "key": "user-123:cat.gif",
        "name": "Funny Cat",
        "status": "public",
        "persist": false,
        "url": "https://storage/user-123:cat.gif",
        "thumbnail_url": "thumbnails/thumb_cat.jpg",
        "download": 5,
        "created_at": "2026-09-08T12:00:00Z",
        "updated_at": "2026-09-08T12:00:00Z"
      }
    ],
    "total": 1
  }
  ```

---

### `GET /gifs/me/{key}`
Retrieve details for a specific GIF.

- **Auth Required**: **Yes**
- **Path Parameters**:
  - `key` (string, required): GIF key.

#### Responses
- **`200 OK`**
  ```json
  {
    "key": "user-123:cat.gif",
    "name": "Funny Cat",
    "status": "public",
    "persist": false,
    "url": "https://storage/user-123:cat.gif",
    "thumbnail_url": "thumbnails/thumb_cat.jpg",
    "download": 5,
    "created_at": "2026-09-08T12:00:00Z",
    "updated_at": "2026-09-08T12:00:00Z"
  }
  ```

---

### `GET /gifs/me/{key}/download`
Obtain a presigned download URL for the GIF.

- **Auth Required**: **Yes** (Owner or authorized share recipient)

#### Responses
- **`200 OK`**
  ```json
  {
    "download_url": "http://127.0.0.1:9000/ffgif-media/user-123%3Acat.gif?X-Amz-Algorithm=..."
  }
  ```

---

### `GET /gifs/me/{key}/thumbnail`
Obtain a presigned MinIO URL for the GIF's thumbnail image.

- **Auth Required**: **Yes**

#### Responses
- **`200 OK`**
  ```json
  {
    "thumbnail_url": "http://127.0.0.1:9000/ffgif-media/thumbnails%2Fthumb_cat.jpg?..."
  }
  ```

---

### `PATCH /gifs/me/{key}`
Update GIF metadata (name, status, persist) with optimistic locking.

- **Auth Required**: **Yes**
- **Headers**:
  - `If-Match` (string, required): Current `updated_at` RFC3339 timestamp.

#### Request Body
```json
{
  "name": "Renamed Cat GIF",
  "status": "private",
  "persist": true
}
```

#### Responses
- **`200 OK`** - Returns updated GIF object.
- **`412 Precondition Failed`** - If `If-Match` header is missing or stale.

---

### `DELETE /gifs/me/{key}`
Delete a GIF and its storage records.

- **Auth Required**: **Yes** (Owner only)

#### Responses
- **`200 OK`** (or `204 No Content`)

---

### `GET /gifs/me/recents`
List the 20 most recent GIFs for the authenticated user.

- **Auth Required**: **Yes**

#### Responses
- **`200 OK`**
  ```json
  [
    {
      "key": "user-123:recent.gif",
      "name": "Recent GIF",
      "status": "public",
      "persist": false,
      "url": "https://storage/user-123:recent.gif",
      "thumbnail_url": "thumbnails/thumb.jpg",
      "download": 0,
      "created_at": "2026-09-08T12:00:00Z"
    }
  ]
  ```

---

### `POST /gifs/me/recents/{key}/save`
Save a GIF to user's persistent recents list.

- **Auth Required**: **Yes**

#### Responses
- **`204 No Content`**

---

## 6. Sharing & Access Control (`/gifs` and `/s`)

### `POST /gifs/me/{key}/shares`
Share an owned GIF with another registered user by email with a custom expiration.

- **Auth Required**: **Yes** (Owner only)
- **Path Parameters**:
  - `key` (string, required): GIF key to share.

#### Request Body
```json
{
  "shared_with": "friend@example.com",
  "expire_at": "2026-09-15T00:00:00Z"
}
```

#### Responses
- **`201 Created`**
  ```json
  {
    "message": "gif shared successfully"
  }
  ```
- **`403 Forbidden`** - Authenticated user is not the owner of the GIF.
- **`404 Not Found`** - Recipient email or GIF does not exist.

---

### `GET /gifs/me/shares`
List all GIFs shared with or shared by the authenticated user.

- **Auth Required**: **Yes**

#### Responses
- **`200 OK`**
  ```json
  [
    {
      "id": "1",
      "name": "Shared GIF",
      "url": "https://storage/gif.gif",
      "thumbnail_url": "thumbnails/thumb.jpg",
      "gif_key": "user-1:gif.gif",
      "owner_id": "owner-uuid",
      "shared_with": "my-user-uuid",
      "expires_at": "2026-09-15T00:00:00Z"
    }
  ]
  ```

---

### `DELETE /gifs/me/{key}/shares/{shareWithId}`
Revoke access to a shared GIF for a specific user.

- **Auth Required**: **Yes** (Owner only)

#### Responses
- **`200 OK`**

---

### `POST /s`
Create a public, tokenized share link with optional email notification.

- **Auth Required**: **Yes** (Owner only)

#### Request Body
```json
{
  "gif_key": "user-123:my_gif.gif",
  "email": "recipient@example.com",
  "expire_at": "2026-09-20T00:00:00Z"
}
```

#### Responses
- **`201 Created`**
  ```json
  {
    "token": "a4f8e91c-7b2d-4f1e-9a3b-8c7d6e5f4a3b"
  }
  ```

---

### `GET /s/{token}`
Retrieve public share details using a share token.

- **Auth Required**: **No** (Public endpoint)
- **Path Parameters**:
  - `token` (string, required): Public share UUID token.

#### Responses
- **`200 OK`**
  ```json
  {
    "gif_key": "user-123:my_gif.gif",
    "name": "Public GIF",
    "url": "https://storage/user-123:my_gif.gif",
    "thumbnail_url": "thumbnails/thumb.jpg",
    "expires_at": "2026-09-20T00:00:00Z",
    "created_at": "2026-09-08T12:00:00Z"
  }
  ```
- **`404 Not Found`** - Token not found or expired.

#### Example Call
```bash
curl -X GET http://localhost:8080/s/a4f8e91c-7b2d-4f1e-9a3b-8c7d6e5f4a3b
```
