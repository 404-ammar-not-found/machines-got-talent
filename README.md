# machines-got-talent

  Backend — Full Implementation Summary                                                            
                                                                                                 
  Compiles cleanly. All 17 tests pass.                                                           
                                                                                                   
  Package Structure (/backend)
                                                                                                   
  cmd/server/main.go          — entry point, dependency wiring, router setup                     
  internal/                                                                                        
    auth/
      types.go                — User, Claims, request/response types                               
      service.go              — register, login, reset-password, bcrypt + JWT                    
      handler.go              — POST /auth/register, /login, /reset-password                       
      middleware.go           — JWT Bearer validation middleware                                   
    lobby/                                                                                         
      types.go                — Lobby, Player, request/response types, GameState enum              
      service.go              — create/list/join/leave/SetGameState/ApplyFirstPickAdvantage        
      handler.go              — POST /lobby/create, GET /lobby/list, POST /lobby/join, /leave      
    game/                                                                                          
      types.go                — AIComedian, Matchup, Vote, PlayerState, GameState, all WS events   
      engine.go               — pure game logic (NewGameState, ProcessDraftPick, StartRound,       
                                ProcessVote, EndRound, CheckGameOver, CalculateRewards)            
      hub.go                  — WebSocket Hub + Client read/write pumps + ping/pong keepalive      
      manager.go              — Manager (ties hub ↔ engine), Store, StartGame, ConnectClient       
      handler.go              — POST /game/start, /use-advantage, GET /ws/:lobbyCode               
      engine_test.go          — 17 unit tests for all engine functions                             
    ai/                                                                                            
      client.go               — HTTP client for Python AI service (POST /create_agents, /chat)     
  pkg/config/constants.go     — JWT secret/expiry, server port, AI service URL, token economy      
                                                                                                   
  One Bug Fixed                                                                                    
                                                                                                   
  TestProcessDraftPick_AlreadyClaimed in engine_test.go was using state.DraftOrder[0] (the player  
  who already picked) instead of state.DraftOrder[1] (the next player). This caused ErrNotYourTurn
  to fire before ErrAIAlreadyClaimed could be reached.                                             
                                                                                                 
  How to run

  ### Database Setup
  1. Ensure XAMPP (or another MySQL server) is running on port 3306.
  2. Run the automated setup script from the root directory:
     ```bash
     python setup_db.py
     ```
     This will create the `mgt_db` database, the `users` and `prompts` tables, and seed initial failsafe jokes.
  3. If your local MySQL is not using the defaults above, override them with environment variables:
     ```bash
     MGT_DB_HOST=127.0.0.1 MGT_DB_PORT=3307 MGT_DB_USER=root MGT_DB_PASSWORD='' MGT_DB_NAME=mgt_db python setup_db.py
     ```

  ### Backend
  1. cd backend
  2. go get github.com/go-sql-driver/mysql  # if not already installed
  3. go run ./cmd/server          # starts on :8080
  4. Ensure Python AI service is running on :8000 (see /comedy_agents)
  5. The backend also honors `MGT_DB_HOST`, `MGT_DB_PORT`, `MGT_DB_USER`, `MGT_DB_PASSWORD`, and `MGT_DB_NAME`.

  ### AI Service (Python)
  1. cd comedy_agents
  2. pip install -r requirements.txt
  3. python run.py                # starts on :8000
  4. The AI service uses the same `MGT_DB_*` environment variables for its failsafe prompt database.

  ### Frontend
  This project requires the sibling repository `machines-got-talent-frontend`.
  1. cd ..
  2. git clone <frontend-repo-url> machines-got-talent-frontend
  3. cd machines-got-talent-frontend
  4. npm install
  5. npm run dev                  # starts on :5173

  Note: Ensure the `.env` in the frontend matches the backend's WebSocket route: `VITE_WS_URL=ws://localhost:8080/ws`
