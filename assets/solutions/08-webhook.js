// Étape 8 — Recevoir un webhook signé (HMAC SHA-256)
const express = require('express');
const crypto = require('crypto');
const app = express();

const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET || 'whsec_demo';
const paiements = [];

// On garde le corps BRUT : la signature est calculée sur les octets exacts
app.post('/webhooks/paiement', express.raw({ type: 'application/json' }), (req, res) => {
  const recue = req.get('X-Signature') || '';
  const attendue = crypto.createHmac('sha256', WEBHOOK_SECRET).update(req.body).digest('hex');
  const ok = recue.length === attendue.length &&
    crypto.timingSafeEqual(Buffer.from(recue), Buffer.from(attendue));
  if (!ok) return res.status(401).json({ error: 'Signature invalide' });

  const evt = JSON.parse(req.body.toString());
  if (!paiements.some(p => p.id === evt.id)) paiements.push(evt); // idempotent
  res.json({ recu: true });
});

app.get('/api/paiements', (req, res) => res.json(paiements));

app.listen(process.env.PORT || 3000);
