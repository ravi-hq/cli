// Package crypto provides end-to-end encryption primitives for the Ravi CLI.
//
// It implements the same cryptographic protocol used by the Ravi backend and
// dashboard so that content encrypted server-side can be decrypted locally on
// the user's machine. The protocol is:
//
//  1. Key derivation: Argon2id(PIN, salt) produces a 32-byte seed.
//  2. Keypair: libsodium-compatible crypto_box_seed_keypair (SHA-512 + clamp)
//     derives a Curve25519 keypair from the seed.
//  3. Encryption: NaCl SealedBox (anonymous sender, X25519-XSalsa20-Poly1305).
//  4. Ciphertext format: "e2e::<base64-ciphertext>".
//
// The package also provides session helpers that prompt the user for their
// 6-digit PIN, derive the keypair, verify it against the server-stored
// verifier, and cache the keypair in memory for the duration of the process.
package crypto
