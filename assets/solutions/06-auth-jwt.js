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

app.post('/auth/login', (req, res) => {
  const u = users.find(x => x.email === req.body.email && x.password === req.body.password);
  if (!u) return res.status(401).json({ error: 'Identifiants invalides' });
  const token = jwt.sign({ sub: u.id, role: u.role }, SECRET, { expiresIn: '15m' });
  res.json({ token });
});

function auth(req, res, next) {
  const [type, token] = (req.headers.authorization || '').split(' ');
  if (type !== 'Bearer' || !token) return res.status(401).json({ error: 'Token manquant' });
  try { req.user = jwt.verify(token, SECRET); next(); }
  catch { res.status(401).json({ error: 'Token invalide' }); }
}

const requireRole = role => (req, res, next) =>
  req.user.role === role ? next() : res.status(403).json({ error: 'Accès refusé' });

app.get('/api/menu', (req, res) => res.json(['Pizza', 'Burger']));          // public
app.get('/api/profil', auth, (req, res) => res.json({ id: req.user.sub, role: req.user.role }));
app.delete('/api/plats/:id', auth, requireRole('admin'), (req, res) => res.status(204).end());

app.listen(process.env.PORT || 3000);
