// Étape 8 — Recevoir un webhook signé (HMAC SHA-256)
// Le « prestataire de paiement » (apilab) envoie un POST avec l'en-tête
// X-Signature = HMAC_SHA256(corps_brut, WEBHOOK_SECRET) en hexadécimal.
const express = require('express');
const crypto = require('crypto');
const app = express();

const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET || 'whsec_demo';
const paiements = [];

// TODO 1 : POST /webhooks/paiement — lire le corps BRUT : express.raw({ type: 'application/json' })
// TODO 2 : recalculer la signature, comparer avec crypto.timingSafeEqual → 401 si invalide
// TODO 3 : parser l'événement et l'enregistrer UNE seule fois par id (idempotence) → 200 { recu: true }

app.get('/api/paiements', (req, res) => res.json(paiements));

app.listen(process.env.PORT || 3000);
