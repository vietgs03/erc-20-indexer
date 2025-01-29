# erc-20-indexer

## Project Description

The `erc-20-indexer` is a tool designed to index and track ERC-20 token transactions on the Ethereum blockchain. It provides a way to monitor and analyze token transfers, balances, and other relevant data for ERC-20 tokens. The project is implemented using Golang and Ethereum.

## Installation

### Prerequisites

- Golang (v1.16 or higher)
- MongoDB (v4 or higher)

### Steps

1. Clone the repository:
   ```sh
   git clone https://github.com/vietgs03/erc-20-indexer.git
   cd erc-20-indexer
   ```

2. Install dependencies:
   ```sh
   go mod tidy
   ```

3. Set up environment variables:
   Create a `.env` file in the root directory and add the following variables:
   ```
   MONGODB_URI=<your_mongodb_uri>
   INFURA_PROJECT_ID=<your_infura_project_id>
   ```

4. Start the application:
   ```sh
   go run main.go
   ```

## Usage

### Example

1. Start the application:
   ```sh
   go run main.go
   ```

2. The application will connect to the Ethereum blockchain and start indexing ERC-20 token transactions. You can monitor the progress in the console output.

3. Access the indexed data:
   You can query the MongoDB database to access the indexed data. For example, you can use a MongoDB client or write scripts to fetch and analyze the data.

### Additional Information

- The application uses the Infura API to connect to the Ethereum blockchain. Make sure you have a valid Infura project ID.
- You can customize the indexing parameters by modifying the configuration file (`config.go`).

