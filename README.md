Blob20-Index

Blob20-Index is an indexer for Blob-20 tokens, which originated from Blobscriptions (Ethscriptions ESIP-8). It is developed using Go and MySQL 8.0. The Blob-20 protocol was created by @wgw_eth, and our indexer fully complies with the standards set by @wgw_eth for the Blob-20 token protocol.

Features
Fully compatible with the Blob-20 token protocol
Developed using Go for excellent performance
Uses MySQL 8.0 for stable and reliable data storage
Provides simple and easy-to-use API interfaces for seamless integration with other applications
Installation

Clone the repository to your local machine:

git clone https://github.com/blob20-index/blob20-index.git

Navigate to the project directory:

cd blob20-index


Install dependencies:

go mod download


Configure the database:

Set up the MySQL database connection information in the config.yaml file.


Run the project:

go run main.go


Initial database snapshot download:

https://github.com/blob20-index/blob20-index/releases/tag/blob20_database_snapshot


Usage
Blob20-Index provides simple and easy-to-use RESTful API interfaces. You can retrieve information about Blob-20 tokens by sending HTTP requests.

API Documentation
For detailed API documentation, please refer to the API Documentation.

Contributing
We welcome contributions in any form, including but not limited to:

Submitting issues
Submitting pull requests
Improving documentation
Providing suggestions and feedback
Acknowledgements
We would like to express our gratitude to @wgw_eth for developing the Blob-20 token protocol. Without his innovation, this project would not have been possible.

We also thank qingmeng, Allin, and other developers for their generous sharing, which has allowed us to better understand and implement the Blob-20 token protocol.

License
This project is licensed under the MIT License.
