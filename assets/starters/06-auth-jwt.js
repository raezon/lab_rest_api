// Étape 6 — Authentification (JWT) et autorisation (rôles)
const express = require('express');
const jwt = require('jsonwebtoken');
const app = express();
app.use(express.json());

const SECRET = process.env.JWT_SECRET || 'dev-secret-a-changer';
const users = [
  { id: 1, email: 'chef@resto.fr', password: 'admin123', role: 'admin' },
  { id: 2, email: 'client@resto.fr', password: 'client123', role: 'user' },
];

// TODO 1 : POST /auth/login { email, password }
//          → 200 { token } signé avec { sub, role }, expiresIn '15m' ; sinon 401

// TODO 2 : middleware auth : lit "Authorization: Bearer <token>", vérifie le JWT,
//          place le payload dans req.user ; sinon 401   (AUTHENTIFICATION)

// TODO 3 : requireRole(role) : 403 si req.user.role ne correspond pas (AUTORISATION)

app.get('/api/menu', (req, res) => res.json(['Pizza', 'Burger']));   // public
// TODO 4 : GET /api/profil            → protégé par auth → { id, role }
// TODO 5 : DELETE /api/plats/:id      → auth + requireRole('admin') → 204

app.listen(process.env.PORT || 3000);
