# Chat Game Lab - Running the Application

## Frontend Only (Design/Text Editing)

For designers and text editors who only need to work on the UI:

```bash
./run-frontend.sh
```

This will:
- Install dependencies if needed
- Start React development server on http://localhost:3000
- **For mock mode (no server needed)**: Open http://localhost:3000?mock=true

### Mock Mode Features
- ✅ Appears logged in automatically  
- ✅ Shows sample games and content
- ✅ All UI navigation works
- ✅ Perfect for design/CSS/text work
- ✅ No backend server required

## Backend Server

For full development with real API:

```bash
./run-server.sh
```

This runs the Go backend server via Docker.

## Full Stack Development

Run both in separate terminals:

```bash
# Terminal 1 - Backend
./run-server.sh

# Terminal 2 - Frontend  
./run-frontend.sh
```

Then open http://localhost:3000 (without `?mock=true`)

## Team Usage

**Designers/Text Editors:**
- Only need: `./run-frontend.sh`
- Open: http://localhost:3000?mock=true
- No server setup required!

**Full Developers:**
- Run both scripts in separate terminals
- Open: http://localhost:3000