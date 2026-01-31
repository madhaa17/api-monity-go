# API Documentation & Progress History

This document tracks the implemented APIs and their specifications.

## Base URL
`http://localhost:8080` (Development)

## Standard Response Format
All API responses follow this standard JSON structure:

### Success
```json
{
  "success": true,
  "message": "Operation successful",
  "data": {
    // Response data goes here
  }
}
```

### Error
```json
{
  "success": false,
  "message": "Error description",
  "errors": "Detailed error message or validation errors object"
}
```

---

## 1. Authentication Module
**Status**: ✅ Implemented

### POST /auth/register
Register a new user.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "registration successful",
  "data": {
    "Token": "eyJhbGciOiJIUzI1Ni...",
    "User": {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "USER",
      "createdAt": "2026-01-31T12:00:00Z",
      "updatedAt": "2026-01-31T12:00:00Z"
    }
  }
}
```

**Errors:**
- `400 Bad Request`: Missing fields or invalid JSON.
- `409 Conflict`: Email already registered.

### POST /auth/login
Login an existing user.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "login successful",
  "data": {
    "Token": "eyJhbGciOiJIUzI1Ni...",
    "User": {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "USER",
      "createdAt": "2026-01-31T12:00:00Z",
      "updatedAt": "2026-01-31T12:00:00Z"
    }
  }
}
```

**Errors:**
- `400 Bad Request`: Missing fields.
- `401 Unauthorized`: Invalid email or password.

---

## 2. Asset Management Module
**Status**: ✅ Implemented

### POST /assets
Create a new asset.
**Headers**: `Authorization: Bearer <token>`
**Request Body**:
```json
{
  "name": "Bitcoin",
  "type": "CRYPTO",
  "quantity": 0.5,
  "symbol": "BTC"
}
```
**Response (201 Created)**

### GET /assets
List all assets for the authenticated user.
**Headers**: `Authorization: Bearer <token>`
**Response (200 OK)**

### GET /assets/{id}
Get a specific asset.
**Headers**: `Authorization: Bearer <token>`
**Response (200 OK)**

### PUT /assets/{id}
Update a specific asset.
**Headers**: `Authorization: Bearer <token>`
**Request Body**:
```json
{
  "quantity": 0.75
}
```
**Response (200 OK)**

### DELETE /assets/{id}
Delete a specific asset.
**Headers**: `Authorization: Bearer <token>`
**Response (200 OK)**
