import { useEffect, useState } from 'react'
import { api, type IntentResponse } from './api'
import './App.css'

const SESSION = 'sess_demo'

export default function App() {
  const [health, setHealth] = useState<string>('checking…')
  const [intent, setIntent] = useState<IntentResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .health()
      .then((h) => setHealth(h.status))
      .catch(() => setHealth('unreachable'))
  }, [])

  async function sendLogin() {
    setError(null)
    try {
      await api.sendEvent({ session_id: SESSION, user_id: 'user_001', type: 'LOGIN' })
      setIntent(await api.intent(SESSION))
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <main className="app">
      <header>
        <h1>Parallax</h1>
        <p className="tagline">Real-time intent inference for digital payments</p>
        <p className="status">
          gateway: <span className={health === 'ok' ? 'ok' : 'bad'}>{health}</span>
        </p>
      </header>

      <section>
        <button onClick={sendLogin}>Send test LOGIN event</button>
        {error && <pre className="error">{error}</pre>}
      </section>

      {intent && (
        <section className="intent">
          <h2>Intent state — {SESSION}</h2>
          <ul>
            {Object.entries(intent.hypotheses).map(([k, v]) => (
              <li key={k}>
                <span>{k}</span>
                <span>{(v * 100).toFixed(0)}%</span>
              </li>
            ))}
          </ul>
          <p>
            uncertainty {intent.uncertainty.toFixed(2)} · action <b>{intent.action}</b>
          </p>
          <p className="evidence">{intent.evidence.join(', ')}</p>
        </section>
      )}
    </main>
  )
}
