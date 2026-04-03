/**
 * Cryptographic utilities for the frontend.
 */

/**
 * Hashes a password with the username as a salt to avoid sending
 * raw passwords over the wire and to protect against simple leaks.
 * 
 * Uses SHA-256 via the browser's native SubtleCrypto API.
 */
export async function hashPassword(username: string, password: string): Promise<string> {
  if (!password) return "";
  
  // Normalize username to lowercase to ensure consistency
  const normalizedUsername = username.toLowerCase().trim();
  
  // Combine password and username
  const data = `${password}:${normalizedUsername}`;
  
  // Encode as UTF-8
  const msgUint8 = new TextEncoder().encode(data);
  
  // Hash the data
  const hashBuffer = await crypto.subtle.digest("SHA-256", msgUint8);
  
  // Convert buffer to hex string
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
  
  return hashHex;
}
