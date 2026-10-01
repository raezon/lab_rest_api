// Étape 9 — Polling vs Server-Sent Events (push serveur)
const express = require('express');
const app = express();
app.use(express.json());

const ETAPES = ['recue', 'en_cuisine', 'prete', 'livree'];
const commande = { id: 42, statut: 'recue' };
const abonnes = new Set(); // les réponses SSE ouvertes

function avancer() {
  const i = ETAPES.indexOf(commande.statut);
  if (i < ETAPES.length - 1) commande.statut = ETAPES[i + 1];
  // TODO 3 : pousser `data: <json>\n\n` à chaque abonné
}

// Polling : le client redemande régulièrement
app.get('/api/commandes/42', (req, res) => res.json(commande));

// TODO 1 : GET /api/commandes/42/events en SSE :
//   en-têtes Content-Type: text/event-stream, Cache-Control: no-cache
//   envoyer l'état courant immédiatement, ajouter res aux abonnés
// TODO 2 : retirer l'abonné quand la connexion se ferme (req.on('close'))

app.post('/api/commandes/42/avancer', (req, res) => { avancer(); res.json(commande); });

app.listen(process.env.PORT || 3000);
