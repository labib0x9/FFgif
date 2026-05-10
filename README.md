# ProjectPDF

ProjectPDF is a API service that accepts images and converts them into a combined pdf.

---

# Features

- User can authenticate.
- Authenticated user can save upto 10pdfs.
- User upload images (each image upto 5mb, at most 10 images), then generate a combined pdf.
- Download the generated pdf

---

# Architecture

    [Client] -> [Frontend] -> [Backend] -> [Client]
                                  |
                                  V
    ----------------------------------------------------------------------------------------------------
    |                                             [BACKEND]                                            |
    |                                                                                                  |
    |               [ROutes]   ->    [           Middlewares        ]                                  |
    |                               /                 |             \                                  |
    |                              V                  V              V                                 |
    |                        [Auth Service]  [Convert Service]  [User Service]                         |
    |                                                |                                                 |
    |                                                V                                                 |
    |                                           [Job Queue]                                            |
    |                                                |                                                 |
    |                                                V                                                 |
    |                   [Object storage]    <-   [Worker pool] (pdf convertion)    ->   [Database]     |
    |                                                                                                  |
    ----------------------------------------------------------------------------------------------------

- Each convertion is executed in isolated environment, worker pool selects container
- PDFs are stored in object storage
- Graceful shutdown
- Image verification(corrupt and malicious files, wrong MIME)

- mailtrap is used to check account verification.
- minio is used for Object Storage
- RabbitMq is userd for message queue
- Redis is used for rate limiting.

---

# API LIST
## Auth APIs
```
POST /auth/login
GET /auth/logout
POST /auth/signup
GET /auth/verify?token=
POST /auth/verify/resend
POST /auth/forgot-password
POST /auth/reset
GET /auth/reset?token=
```

## User APIs
```
GET /users/profile/me
POST /users/profile
POST /users/change-password
```

## Upload APIs
```
POST /uploads
GET /uploads/{key}/status
```

## GIFs APIs
```
GET /gifs
GET /gifs/{id}
GET /gifs/{id}/download
DELETE /gifs/{id}
```

## Convert APIs
```
POST /convert
GET /convert/status/{jobId}
```

## Admin APIs
```
GET /admin/users
DELETE /admin/users/{id}
```

```
cp .env.example .env
```