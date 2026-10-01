// Étape 10 — Architecture événementielle : Pub/Sub en mémoire
const express = require('express');
const { EventEmitter } = require('events');
const app = express();
app.use(express.json());

const bus = new EventEmitter();   // joue le rôle du broker (Kafka, RabbitMQ…)
const notifications = [];
const factures = [];
let nextId = 1;

// TODO 1 : abonné « notifications » sur 'commande.creee'
//          → pousse le texte `Commande <id> reçue pour <client>`
// TODO 2 : abonné « facturation » sur 'commande.creee'
//          → pousse { commandeId, montant }

// TODO 3 : POST /api/commandes { client, montant }
//          → 400 si invalide ; sinon crée la commande, PUBLIE l'événement, 201
//          Le producteur ne doit appeler AUCUN service directement.

app.get('/api/notifications', (req, res) => res.json(notifications));
app.get('/api/factures', (req, res) => res.json(factures));

app.listen(process.env.PORT || 3000);
