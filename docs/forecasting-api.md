# Forecasting API

Read-only endpoints for serving forecast data produced by the Python forecasting pipeline.

All endpoints require JWT authentication via `Authorization: Bearer <token>` header or `access_token` cookie.

---

## List Forecasts

```
GET /api/forecasts/?forecast_month=YYYY-MM
```

Returns all forecasts for the authenticated user for the given month.

### Query Parameters

| Parameter | Required | Format | Description |
|-----------|----------|--------|-------------|
| `forecast_month` | Yes | `YYYY-MM` | Target month (e.g., `2026-03`) |

### Response `200 OK`

```json
{
  "count": 2,
  "results": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "recurrencyPatternId": 42,
      "type": "DEBIT",
      "recurrencyType": "recurrent_fixed",
      "forecastAmount": 1500.00,
      "forecastLow": 1400.00,
      "forecastHigh": 1600.00,
      "forecastDate": "2026-03-05T00:00:00Z",
      "forecastMonth": "2026-03-01T00:00:00Z",
      "cousin": 123,
      "cousinName": "Imobiliaria XYZ",
      "category": "Moradia",
      "description": "Aluguel",
      "accountId": "acc-uuid-here"
    },
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "type": "DEBIT",
      "recurrencyType": "recurrent_variable",
      "forecastAmount": 850.00,
      "forecastLow": 600.00,
      "forecastHigh": 1100.00,
      "forecastMonth": "2026-03-01T00:00:00Z",
      "cousin": 456,
      "cousinName": "Supermercado ABC",
      "category": "Mercado",
      "description": "Supermercado ABC - Compras semanais",
      "accountId": "acc-uuid-here"
    }
  ]
}
```

### Errors

| Status | Condition |
|--------|-----------|
| `400` | Missing or invalid `forecast_month` parameter |
| `401` | Missing or invalid JWT |

---

## Get Forecast by UUID

```
GET /api/forecasts/{uuid}
```

Returns a single forecast by its UUID.

### Path Parameters

| Parameter | Description |
|-----------|-------------|
| `uuid` | Forecast UUID (the `id` field from list response) |

### Response `200 OK`

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "recurrencyPatternId": 42,
  "type": "DEBIT",
  "recurrencyType": "recurrent_fixed",
  "forecastAmount": 1500.00,
  "forecastLow": 1400.00,
  "forecastHigh": 1600.00,
  "forecastDate": "2026-03-05T00:00:00Z",
  "forecastMonth": "2026-03-01T00:00:00Z",
  "cousin": 123,
  "cousinName": "Imobiliaria XYZ",
  "category": "Moradia",
  "description": "Aluguel",
  "accountId": "acc-uuid-here"
}
```

### Errors

| Status | Condition |
|--------|-----------|
| `400` | Missing UUID path parameter |
| `401` | Missing or invalid JWT |
| `404` | Forecast not found or doesn't belong to user |

---

## Response Fields

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | string | No | UUID identifier |
| `recurrencyPatternId` | integer | Yes | FK to recurrency_patterns (null for historical forecasts) |
| `type` | string | No | `"DEBIT"` or `"CREDIT"` |
| `recurrencyType` | string | No | `"recurrent_fixed"`, `"recurrent_variable"`, or `"irregular"` |
| `forecastAmount` | number | No | Point estimate (always positive) |
| `forecastLow` | number | Yes | Lower bound of confidence band |
| `forecastHigh` | number | Yes | Upper bound of confidence band |
| `forecastDate` | string (ISO 8601) | Yes | Predicted date (set for `recurrent_fixed` only) |
| `forecastMonth` | string (ISO 8601) | No | 1st of the target month |
| `cousin` | integer | Yes | Counterparty ID for similar transactions |
| `cousinName` | string | Yes | Human-readable merchant name |
| `category` | string | Yes | Transaction category |
| `description` | string | Yes | Human-readable forecast label |
| `accountId` | string | No | FK to accounts |

Nullable fields are omitted from the response when null.

---

## Recurrency Types

| Type | Meaning | `forecastDate` | `forecastAmount` represents |
|------|---------|----------------|-----------------------------|
| `recurrent_fixed` | ~1x/month, predictable day (rent, salary) | Set | Single transaction amount |
| `recurrent_variable` | Multiple times/month (groceries, transport) | Null | Monthly aggregate total |
| `irregular` | Spending envelope by category | Null | Expected monthly total |
