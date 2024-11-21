
## Prerequisites

Make sure you have the following installed:

- [Node.js](https://nodejs.org/) (for frontend)
- [Go](https://go.dev/dl/) (for backend)
- [Docker](https://www.docker.com/) (optional for Redis setup)
- [Redis](https://redis.io/) (either local or remote instance)

## Getting Started

Follow these steps to get the project up and running on your local machine.

### 1. Clone the Repository

Clone the repository from GitHub:

```bash
git clone https://github.com/Rajatbisht12/EmiterA.git
cd EmiterA
cd server
# Create a .env file inside the server folder and add the following environment variables
nano .env
'''
REDIS_URL=redis://default:DaCPjAdf6O1WplQ9mXJPwkcKNEO09fca@redis-13966.c264.ap-south-1-1.ec2.redns.redis-cloud.com:13966/0
PORT=8080
ALLOWED_ORIGINS=*
'''
#Run the Go server: 
'go run main.go'

#The Go server should now be running on http://localhost:8080

# Now Navigate to the ui folder

cd ui
# Create a .env file inside the ui folder and add the following environment variables
nano .env
'''
API_URL=http://localhost:8080/api
WS_URL=ws://localhost:8080
'''

# Install the frontend dependencies

npm instal

# Start the React development server

npm start

# Also you can check the live applicatin by visiting the site: https://emitterarajat.netlify.app/

```