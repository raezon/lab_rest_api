// Étape 9 — Polling vs Server-Sent Events (push serveur)
const express = require('express');
const app = express();
app.use(express.json());

const ETAPES = ['recue', 'en_cuisine', 'prete', 'livree'];
const commande = { id: 42, statut: 'recue' };
const abonnes = new Set();

function avancer() {
  const i = ETAPES.indexOf(commande.statut);
  if (i < ETAPES.length - 1) commande.statut = ETAPES[i + 1];
  for (const res of abonnes) res.write(`data: ${JSON.stringify(commande)}\n\n`);
}

// Polling : le client redemande régulièrement
app.get('/api/commandes/42', (req, res) => res.json(commande));

// SSE : le serveur pousse chaque changement
app.get('/api/commandes/42/events', (req, res) => {
  res.set({ 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache', Connection: 'keep-alive' });
  res.flushHeaders();
  res.write(`data: ${JSON.stringify(commande)}\n\n`);
  abonnes.add(res);
  req.on('close', () => abonnes.delete(res));
});

// Simule la cuisine : fait avancer la commande
app.post('/api/commandes/42/avancer', (req, res) => { avancer(); res.json(commande); });

app.listen(process.env.PORT || 3000);
