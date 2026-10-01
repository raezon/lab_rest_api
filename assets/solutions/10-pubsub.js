// Étape 10 — Architecture événementielle : Pub/Sub en mémoire
const express = require('express');
const { EventEmitter } = require('events');
const app = express();
app.use(express.json());

const bus = new EventEmitter();            // le « broker »
const notifications = [];
const factures = [];
let nextId = 1;

// Abonnés : ils ne connaissent pas le producteur
bus.on('commande.creee', cmd => notifications.push(`Commande ${cmd.id} reçue pour ${cmd.client}`));
bus.on('commande.creee', cmd => factures.push({ commandeId: cmd.id, montant: cmd.montant }));

// Producteur : publie l'événement, sans appeler les services
app.post('/api/commandes', (req, res) => {
  const { client, montant } = req.body;
  if (!client || typeof montant !== 'number') return res.status(400).json({ error: 'client et montant requis' });
  const cmd = { id: nextId++, client, montant };
  bus.emit('commande.creee', cmd);
  res.status(201).json(cmd);
});

app.get('/api/notifications', (req, res) => res.json(notifications));
app.get('/api/factures', (req, res) => res.json(factures));

app.listen(process.env.PORT || 3000);
