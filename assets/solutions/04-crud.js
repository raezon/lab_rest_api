// Étape 4 — CRUD REST complet sur /api/plats
const express = require('express');
const app = express();
app.use(express.json());

let plats = [{ id: 1, nom: 'Pizza', prix: 12.5 }];
let nextId = 2;

app.get('/api/plats', (req, res) => res.json(plats));

app.get('/api/plats/:id', (req, res) => {
  const p = plats.find(x => x.id === Number(req.params.id));
  if (!p) return res.status(404).json({ error: 'Plat introuvable' });
  res.json(p);
});

app.post('/api/plats', (req, res) => {
  const { nom, prix } = req.body;
  if (!nom || typeof prix !== 'number') return res.status(400).json({ error: 'nom et prix requis' });
  const p = { id: nextId++, nom, prix };
  plats.push(p);
  res.status(201).location(`/api/plats/${p.id}`).json(p);
});

app.put('/api/plats/:id', (req, res) => {
  const i = plats.findIndex(x => x.id === Number(req.params.id));
  if (i < 0) return res.status(404).json({ error: 'Plat introuvable' });
  const { nom, prix } = req.body;
  if (!nom || typeof prix !== 'number') return res.status(400).json({ error: 'nom et prix requis' });
  plats[i] = { id: plats[i].id, nom, prix };
  res.json(plats[i]);
});

app.delete('/api/plats/:id', (req, res) => {
  const avant = plats.length;
  plats = plats.filter(x => x.id !== Number(req.params.id));
  if (plats.length === avant) return res.status(404).json({ error: 'Plat introuvable' });
  res.status(204).end();
});

app.listen(process.env.PORT || 3000);
