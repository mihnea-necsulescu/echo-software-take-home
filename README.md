# Senior Software Engineer - Take-Home Exercise: FireGo Wallet Service

## Overview

FireGo Wallet is a simple cryptocurrency wallet management service built in Go that integrates with the Fireblocks API to provide vault-based wallet operations. It acts as a bridge between client applications and Fireblocks, providing a simplified API layer by exposing the following RESTful endpoints:
1. Create Wallet `POST /wallets`
    
   Request Body: 
   ```json
    {
        "name": "<string>"
    }
    ```
   The `Create Wallet` endpoint creates a "link" between a local FireGo wallet and a Fireblocks Vault Account. It calls the `Create a new vault account` Fireblocks API (`POST https://api.fireblocks.io/v1/v1/vault/accounts`) to create a new vault account with the provided name, and then creates a new local wallet with a generated UUID and the `VaultAccountID` set to the one received from Fireblocks. 
    
    Sample response:
    ```json
    {
      "id": "1e0c7297-71fd-42d9-8373-3f025f4f2ef0",
      "name": "test",
      "vaultAccountID": "86"
    }
   ```
   
   _Known limitation_: With the current implementation, if the database save fails after the Fireblocks vault creation, the vault account may be orphaned.

        
2. Get Wallet Balance `GET /wallets/{walletId}/assets/{assetId}/balance`

    The `Get Wallet Balance` endpoint retrieves the balance details for a given wallet and a provided asset. It first queries the database to retrieve the internal wallet, gets the wallet's VaultAccountID, and then uses it to call the `Get the asset balance for a vault account` Fireblocks API (`GET https://api.fireblocks.io/v1/vault/accounts/{vaultAccountId}/{assetId}`) along with the provided AssetID. 

    Sample response:
    ```json
    {
      "id": "BTC_TEST",
      "total": "0.00004",
      "balance": "0.00004",
      "available": "0.00004",
      "pending": "0",
      "frozen": "0",
      "lockedAmount": "0",
      "staked": "0"
    }
    ```
3. Get Deposit Address `GET /wallets/{walletId}/assets/{assetId}/address`

   The `Get Deposit Address` endpoint retrieves the first address for a wallet's asset. It first queries the database to retrieve the internal wallet, gets the wallet's VaultAccountID, and then uses it to call the `Get asset addresses` Fireblocks API (`GET https://api.fireblocks.io/v1/vault/accounts/{vaultAccountId}/{assetId}/addresses_paginated`) along with the provided AssetID.

    Sample response:
    ```json
    {
      "assetId": "BTC_TEST",
      "address": "tb1qchrsjtj6xu6trnfr6d39m3ldcrwta3sq0vj3rm",
      "addressFormat": "SEGWIT",
      "type": "Permanent"
    }
    ```
4. Initiate Transfer `POST /wallets/{walletId}/transactions`

    Request body sample:
    ```json
   {
      "assetId": "BTC_TEST",
      "amount": "0.001",
      "destinationAddress": "tb1qchrsjtj6xu6trnfr6d39m3ldcrwta3sq0vj3rm",
      "note": "Optional Note"
   }
   ```
    The `Initiate Transfer` endpoint attempts to create a Fireblocks transaction with the provided data (after checking if the wallet's current balance is higher than the transfer amount) by calling the `Create a new transaction` Fireblocks API (`POST https://api.fireblocks.io/v1/transactions`) with the following request body:
    ```json
   {
      "operation": "TRANSFER",
      "assetId": "<asset_id>",
      "source": {
        "type": "VAULT_ACCOUNT",
        "id": "<vault_account_id>"
      },
      "destination": {
        "type": "ONE_TIME_ADDRESS",
        "oneTimeAddress": {
          "address": "<destination_address>"
        }
      },
      "amount": "<amount>",
      "note": "<optional_note>"
    }
   ```
   Sample response:
   ```json
   {
      "transactionId": "81424601-6483-4c15-bd40-93aec6f871ed",
      "status": "PENDING_AML_SCREENING",
      "assetId": "BTC_TEST",
      "amount": "0.0000001",
      "destinationAddress": "tb1qlj64u6fqutr0xue85kl55fx0gt4m4urun25p7q",
      "note": "Test transfer"
    }
   ```

    _Known limitation_: Even if the API integration is successful, after being submitted, all transactions end up in a `BLOCKED` status (`BLOCKED_BY_POLICY` substatus).

## Assumptions, Design Choices & Limitations

### Database & Storage
- **Minimal metadata storage**: Only essential wallet information (local ID, name, vault account ID, timestamps) is stored locally; transaction history and detailed asset information remain in Fireblocks. However, a `transactions`/`transfers` table could be added for auditing purposes, but this would also require registering webhooks so that our local transactions' statuses are correctly updated, ensuring they are always in a proper state.

### Fireblocks Integration
- **Asset Wallet Pre-creation**: Asset-specific wallets (e.g., BTC_TEST) must be created separately in Fireblocks before performing balance, address, or transfer operations.
- **Vault Account Model**: All wallets are Fireblocks vault accounts only.
- **First Address Selection**: For deposit addresses, the first available address is returned.

### Minimal Dependencies
- **GORM**: PostgreSQL ORM for database operations and migrations
- **golang-jwt/jwt**: JWT token signing for Fireblocks authentication
- **google/uuid**: Nonce generation for JWT authentication
- **testify**: Testing assertions

### Security
- **Environment Variable Security**: Fireblocks API credentials stored in environment variables provide sufficient security for our scope.
- **No API authentication**: The REST endpoints have no authentication layer (which is acceptable for our scope).

### Error Handling
- **Generic Error Messages**: Error responses provide user-friendly messages rather than exposing internal Fireblocks error details.
- **Vague Error Messages**: Even if they are user-friendly, some of them are too vague (e.g. "Invalid request" for any 4xx status code). Fireblocks error codes should be mapped to specific error messages.
- **Plain Text Error Messages**: For the sake of simplicity, all errors are returned as plain text (acceptable for our scope). They would, however, have to be transformed to JSON responses for consumers.

### Technical Architecture
- **Missing service layer**: For the sake of simplicity, and to avoid over-engineering, the service layer was skipped. The business logic, being relatively simple, is handled directly in each handler. Should it evolve, a service layer would need to be extracted and tested separately.
- **Standard Library HTTP**: Go's `net/http` is sufficient for our scope.
- **In-Memory Configuration**: Application configuration loaded at start-up only.
- **Simple logging**: Basic log output is sufficient for our scope.
- **Docker for Database Only**: Application runs natively while only PostgreSQL is containerized for simplified development. 
- **Repository Layer Testing**: Given the minimal CRUD operations, unit tests were focused on the handler layer where business logic resides and on the Fireblocks client correctness.
- **Missing Idempotency**: No idempotency key support for create/transfer operations, presenting risks for duplicate operations (acceptable for assignment scope)

### Concurrency Considerations
- **HTTP Server Concurrency**: The standard `net/http` server handles concurrent requests automatically
- **Single-Operation Endpoints**: Current endpoints perform sequential operations (DB lookup -> Fireblocks API) where concurrency wouldn't provide benefits
- **Future Enhancements**: Concurrency would be valuable for operations outside our scope (e.g., bulk wallet creation)

### Retry Limitations
- **No Retry Logic**: The service performs no automatic retries for failed operations
- **No Circuit Breaker**: No protection against cascading failures when Fireblocks API is degraded
- **Timeout Handling**: Basic HTTP client timeouts (30s) but no timeout strategies per operation type

## Setup and Testing

### Prerequisites
- Go 1.24.4
- Docker
- Make
- Postman

### Environment Variables
First things first, a `.env` file has to be set up. An `.env.example` is provided with default values that can be copied/renamed to a `.env` file. The only needed setup here is:
- `FIREBLOCKS_API_KEY`: the `credential` from the `Fireblocks Testnet API key` 1password vault
- the `private key` from the `Fireblocks testnet private key` 1password vault has to be copied (following proper formatting) in a `fireblocks_secret.key` file at the project's root level 

### Building, Running & Database Setup
A `makefile` is provided to make everything simple and seamless. Available commands can be seen using `make help`. Running the service for the first time would require running the following commands:
```bash
make setup
make db-up
make run
```
Unit tests can also be run by running either `make test` or `make test-verbose` for verbose output.

### Testing
A Postman collection that can be imported is provided for easy testing.

_Note_: As previously mentioned, asset-specific Fireblocks vault wallets have to be created for any newly created local wallets (their underlying vault accounts) before testing the balance, address, and transfer endpoints. This can be done by calling the `Create a new vault wallet` Fireblocks API (`POST https://api.fireblocks.io/v1/vault/accounts/{vaultAccountId}/{assetId}`, where the `vaultAccountId` is the vault account ID returned by the `Create a Wallet` endpoint, and the `assetID` is the desired asset ID). To properly test the balance and transfer endpoints, this wallet must also be topped up. This can be done by using the `Get Deposit Address` endpoint to fetch the wallet's deposit address, which can then be used with any Testnet Faucet (e.g. https://coinfaucet.eu/en/btc-testnet/, https://bitcoinfaucet.uo1.net/send.php). 
