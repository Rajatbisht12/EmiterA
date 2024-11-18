// store.js
import { configureStore, createSlice } from '@reduxjs/toolkit';

const gameSlice = createSlice({
  name: 'game',
  initialState: {
    username: '',
    isLoggedIn: false,
    gameId: null,
    deck: [],
    hasDefuse: false,
    gameStatus: null,
    leaderboard: [], // Initialize as empty array instead of null
    lastDrawnCard: null
  },
  reducers: {
    setUsername: (state, action) => {
      state.username = action.payload;
      state.isLoggedIn = true;
    },
    setGameState: (state, action) => {
      state.gameId = action.payload.id;
      state.deck = action.payload.deck;
      state.hasDefuse = action.payload.hasDefuse;
    },
    updateLeaderboard: (state, action) => {
      state.leaderboard = action.payload || []; // Ensure we never set null
    },
    setLastDrawnCard: (state, action) => {
      state.lastDrawnCard = action.payload;
    },
    setGameStatus: (state, action) => {
      state.gameStatus = action.payload;
    }
  }
});

export const { 
  setUsername, 
  setGameState, 
  updateLeaderboard, 
  setLastDrawnCard, 
  setGameStatus 
} = gameSlice.actions;

export const store = configureStore({
  reducer: {
    game: gameSlice.reducer
  }
});