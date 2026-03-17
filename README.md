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
  
  cd backend                                                                                       
  go run ./cmd/server          # starts on :8080                                                 
  # Python AI service must be running on :8000   