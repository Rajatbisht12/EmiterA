import React, { useState, useEffect, useCallback } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { Trophy, Bomb, Cat, Shield, Shuffle } from 'lucide-react';
import { 
  setUsername, 
  setGameState, 
  updateLeaderboard, 
  setLastDrawnCard, 
  setGameStatus 
} from './store';
import './Game.css';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';
const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080';

const Game = () => {
  const dispatch = useDispatch();
  const gameState = useSelector((state) => state.game);
  const [message, setMessage] = useState('');
  const [playerScores, setPlayerScores] = useState(new Map());
  const [ws, setWs] = useState(null);

  // Define resumeGame with useCallback to prevent recreation on every render
  const resumeGame = useCallback(async (gameId) => {
    try {
      const response = await fetch(`${API_URL}/game/resume`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ gameId }),
      });
      
      const game = await response.json();
      dispatch(setGameState(game));
      setMessage('Game resumed!');
    } catch (error) {
      setMessage('Error resuming game.');
    }
  }, [dispatch]);

  // Initialize WebSocket connection
  useEffect(() => {
    if (gameState.isLoggedIn && !ws) {
      const websocket = new WebSocket(`${WS_URL}/ws?username=${gameState.username}`);
      
      websocket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type === 'score_update') {
          setPlayerScores(prev => {
            const newScores = new Map(prev);
            newScores.set(data.username, {
              current: data.score,
              previous: data.previous
            });
            return newScores;
          });
        }
      };

      setWs(websocket);
      return () => websocket.close();
    }
  }, [gameState.isLoggedIn, gameState.username, ws]);

  // Load game state from session storage on component mount
  useEffect(() => {
    const savedGame = sessionStorage.getItem('gameState');
    if (savedGame) {
      const parsedGame = JSON.parse(savedGame);
      dispatch(setGameState(parsedGame));
      resumeGame(parsedGame.gameId);
    }
  }, [dispatch, resumeGame]);

  // Save game state to session storage whenever it changes
  useEffect(() => {
    if (gameState.gameId) {
      sessionStorage.setItem('gameState', JSON.stringify(gameState));
    }
  }, [gameState]);

  // Fetch leaderboard periodically
  useEffect(() => {
    const fetchLeaderboard = async () => {
      try {
        const response = await fetch(`${API_URL}/leaderboard`);
        const data = await response.json();
        dispatch(updateLeaderboard(data));
      } catch (error) {
        console.error('Error fetching leaderboard:', error);
      }
    };

    fetchLeaderboard();
    const interval = setInterval(fetchLeaderboard, 5000);
    return () => clearInterval(interval);
  }, [dispatch]);

  const getCardIcon = (cardType) => {
    switch (cardType) {
      case 'cat':
        return <Cat className="icon" style={{ color: '#f97316' }} />;
      case 'defuse':
        return <Shield className="icon" style={{ color: '#3b82f6' }} />;
      case 'bomb':
        return <Bomb className="icon" style={{ color: '#ef4444' }} />;
      case 'shuffle':
        return <Shuffle className="icon" style={{ color: '#a855f7' }} />;
      default:
        return null;
    }
  };

  const startNewGame = async () => {
    try {
      const response = await fetch(`${API_URL}/game/new`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username: gameState.username }),
      });
      
      const game = await response.json();
      dispatch(setGameState(game));
      setMessage('Game started! Draw a card.');
    } catch (error) {
      setMessage('Error starting game.');
    }
  };
  
  const drawCard = async () => {
    if (!gameState.gameId) return;

    try {
      const response = await fetch(`${API_URL}/game/draw`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ gameId: gameState.gameId }),
      });
      const result = await response.json();
      
      dispatch(setGameState(result.game));
      dispatch(setLastDrawnCard(result.card));
      dispatch(setGameStatus(result.status));

      switch (result.status) {
        case 'defused':
          setMessage('Bomb defused! You can continue playing.');
          break;
        case 'exploded':
          setMessage('BOOM! Game Over!');
          break;
        case 'won':
          setMessage('Congratulations! You won!');
          break;
        case 'shuffled':
          setMessage('Deck shuffled! New game started.');
          break;
        default:
          setMessage('Card drawn! Keep playing.');
      }
    } catch (error) {
      setMessage('Error drawing card.');
    }
  };

  const handleLogin = (e) => {
    e.preventDefault();
    const username = e.target.username.value;
    dispatch(setUsername(username));
  };

  const renderScore = (player) => {
    const scoreInfo = playerScores.get(player.username);
    if (!scoreInfo) {
      return <span>{player.score} points</span>;
    }

    const rawScoreDiff = scoreInfo.current - scoreInfo.previous;
    const scoreDiff = Math.abs(rawScoreDiff);
    const diffColor = rawScoreDiff > 0 ? 'text-green-500' : 'text-red-500';
    
    return (
      <div className="score-container">
        <span>{scoreInfo.current} points</span>
        {scoreDiff !== 0 && (
          <span className={`score-diff ${diffColor}`}>
            ({scoreDiff})
          </span>
        )}
      </div>
    );
  };

  const renderLeaderboard = () => (
    <div className="card leaderboard-card">
      <div className="card-header">
        <h2 className="card-title">
          <Trophy className="icon" style={{ color: '#eab308' }} />
          Leaderboard
        </h2>
      </div>
      <div className="card-content">
        <div className="leaderboard-list">
          {gameState.leaderboard && gameState.leaderboard.length > 0 ? (
            gameState.leaderboard.map((player, index) => (
              <div key={index} className="leaderboard-item">
                <span>{player.username}</span>
                {renderScore(player)}
              </div>
            ))
          ) : (
            <div className="leaderboard-item">
              <span>No scores yet</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );

  if (!gameState.isLoggedIn) {
    return (
      <div className="login-container">
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Welcome to Exploding Kittens</h2>
          </div>
          <div className="card-content">
            <form onSubmit={handleLogin} className="login-form">
              <input
                type="text"
                name="username"
                placeholder="Enter your username"
                className="input"
                required
              />
              <button type="submit" className="button button-primary">
                Start Playing
              </button>
            </form>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="game-container">
      <div className="game-content">
        <div className="game-header">
          <h1 className="game-title">Exploding Kittens</h1>
          <div className="player-info">
            <span className="player-name">Player: {gameState.username}</span>
          </div>
        </div>

        <div className="card game-card">
          <div className="card-content">
            {!gameState.gameId ? (
              <button 
                onClick={startNewGame} 
                className="button button-primary button-full"
              >
                Start New Game
              </button>
            ) : (
              <div className="game-status">
                <div className="status-bar">
                  <span>Cards remaining: {gameState.deck.length}</span>
                  <span>Has Defuse: {gameState.hasDefuse ? 'Yes' : 'No'}</span>
                </div>
                
                <div className="draw-section">
                  <button 
                    onClick={drawCard}
                    disabled={!gameState.deck.length}
                    className="button button-primary button-large"
                  >
                    Draw Card
                  </button>
                </div>

                {message && (
                  <div className="alert">
                    <p>{message}</p>
                  </div>
                )}

                {gameState.lastDrawnCard && (
                  <div className="drawn-card">
                    <div className="card-display">
                      {getCardIcon(gameState.lastDrawnCard.type)}
                      <div className="card-type">
                        {gameState.lastDrawnCard.type.toUpperCase()}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {renderLeaderboard()}
      </div>
    </div>
  );
};

export default Game;