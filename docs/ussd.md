# Parallax USSD Security Architecture: Intent Inference for Unstructured Supplementary Service Data

## 1. Context and Problem Statement

In Nigeria and broader sub-Saharan African financial ecosystems, USSD (Unstructured Supplementary Service Data) remains the predominant digital payment rail for financial inclusion. Initiated via carrier shortcodes (e.g., `*737#` for GTBank, `*894#` for FirstBank, `*901#` for Access Bank), USSD transactions process trillions of Naira annually across feature phones and smartphones alike without requiring internet connectivity or mobile data.

However, USSD represents the most vulnerable attack vector in digital banking:

1. **Absence of Client Telemetry**: Unlike mobile applications or web browsers, USSD provides no device fingerprinting, no canvas rendering, no IP geolocation, no biometric indicators, and no device hardware identifiers.
2. **SIM Swap Fraud**: Attackers bribe telecommunications agents or exploit identity theft to perform unauthorized SIM swaps. The attacker then places the stolen MSISDN (phone number) into a burner device, dials the bank's USSD shortcode, resets the USSD PIN using leaked Personally Identifiable Information (such as BVN or date of birth), and drains the customer's savings via rapid transfers.
3. **Severe Network Constraints**: USSD sessions run on a strict 20 to 30 second session timer managed by the carrier's Home Location Register (HLR) and USSD Gateway. If an external security engine takes longer than 500 milliseconds, the session times out, failing the transaction.
4. **Coercion and Impersonation**: Feature phone users are disproportionately targeted by vishing (voice phishing) and impersonation scams, where fraudsters instruct victims to dial specific USSD string combinations under the guise of "reversing an error" or "updating BVN records".

Traditional banking rules cannot address this nuance: blocking any high-value USSD transfer alienates rural and unbanked customers, while allowing any transaction authorized by PIN enables rampant fraud.

Parallax adapts its **latent intent inference engine** to the USSD environment through telephony metadata, behavioral timing dynamics, and USSD-native intent probing.

---

## 2. End-to-End System Architecture

```text
+-------------------------------------------------------------------------------+
|                             CUSTOMER MOBILE HANDSET                           |
|                    (Feature Phone / Dual-SIM / GSM Terminal)                  |
+-------------------------------------------------------------------------------+
                                        |
                                        | USSD String: *737*1*Amount*Account#
                                        v
+-------------------------------------------------------------------------------+
|                    TELECOMMUNICATIONS INFRASTRUCTURE                          |
|         (MTN Nigeria / Airtel Nigeria / Globacom / 9mobile)                   |
|                                                                               |
|   +-----------------------+               +-------------------------------+   |
|   | Base Transceiver (BTS)| ------------> | Mobile Switching Centre (MSC) |   |
|   +-----------------------+               +-------------------------------+   |
|                                                           |                   |
|                                                           v                   |
|                                           +-------------------------------+   |
|                                           |     Carrier USSD Gateway      |   |
|                                           |     (MAP / SIGTRAN / SMPP)    |   |
|                                           +-------------------------------+   |
+-------------------------------------------------------------------------------+
                                        |
                                        | SMPP / HTTPS XML Protocol
                                        v
+-------------------------------------------------------------------------------+
|                    BANK USSD AGGREGATOR & GATEWAY LAYER                       |
|           (Session State Manager, Menu Flow Engine, Protocol Translator)      |
+-------------------------------------------------------------------------------+
                                        |
                                        | Event Ingestion: POST /v1/events
                                        v
+-------------------------------------------------------------------------------+
|                       PARALLAX INTENT INFERENCE ENGINE                        |
|                                                                               |
|   +-----------------------+               +-------------------------------+   |
|   |  SIM Swap Status API  |               |    USSD Behavioral Baseline   |   |
|   |  (Telco NCC Registry) |               |  (Dwell Time, Velocity, Menu) |   |
|   +-----------------------+               +-------------------------------+   |
|               |                                           |                   |
|               +-------------------+   +-------------------+                   |
|                                   v   v                                       |
|                       +-------------------------------+                       |
|                       |    In-Process GBDT & Markov   |                       |
|                       |    Latent Intent Engine       |                       |
|                       +-------------------------------+                       |
|                                       |                                       |
|                                       v                                       |
|                       +-------------------------------+                       |
|                       |   Decoupled Policy Decider    |                       |
|                       +-------------------------------+                       |
|                                       |                                       |
|                       +---------------+---------------+                       |
|                       |                               |                       |
|                       v                               v                       |
|               [PROBE REQUIRED]                [ALLOW / BLOCK]                 |
|                       |                               |                       |
+-----------------------|-------------------------------|-----------------------+
                        |                               |
                        v                               v
             +--------------------+          +--------------------+
             | Return USSD Probe  |          | Forward to Core    |
             | Interactive Menu   |          | Banking System     |
             | (182 Character UI) |          | (Finacle/Flexcube) |
             +--------------------+          +--------------------+
```

---

## 3. Signal Extraction in Low-Telemetry Environments

Where mobile applications provide rich telemetry, Parallax constructs an evidentiary foundation for USSD using four telephony and interaction vectors:

### 3.1 Telco SIM Swap Verification Signaling

Through integration with national telecommunications clearinghouses (such as the Nigerian Communications Commission SIM registry or direct Mobile Network Operator APIs), Parallax queries the subscriber's SIM lifecycle metadata during session initialization:

1. **IMSI Change Timestamp**: Determines whether the International Mobile Subscriber Identity associated with the MSISDN was altered within the preceding 72 hours.
2. **SIM Age in Handset**: Detects whether the SIM was freshly paired with an unfamiliar IMEI (handset hardware ID).
3. **Roaming Status**: Confirms whether the cellular session originated from an abnormal roaming cell tower.

**Inference Rule**: An IMSI alteration within 48 hours coupled with a USSD PIN change and immediate transfer attempt sets the Account Takeover hypothesis to $P(\text{ATO}) \ge 0.96$, triggering an immediate `BLOCK`.

### 3.2 USSD Dwell Time and Keystroke Dynamics

Because USSD sessions are interactive dialogs mediated by the carrier, the timestamp of each menu transmission is tracked to millisecond precision:

1. **Menu Dwell Time**: The elapsed time between menu display and subscriber input submission.
   - *Scripted / Attack Behavior*: Attackers automating USSD drains via GSM modems or scripts submit inputs in under 800 milliseconds.
   - *Human Baseline*: Normal human users reading a prompt require 3.5 to 8.0 seconds.
   - *Coerced / Manipulated Behavior*: Victims receiving phone call instructions exhibit prolonged hesitations (18 to 25 seconds) approaching session timeout.
2. **Keypad Cadence Deviation**: Deviation from the customer's historical dialling speed during PIN and amount entry.

### 3.3 Session Trajectory and Velocity

Parallax models USSD session states as an ordered trajectory:

```text
USSD_SESSION_START
       |
       v
BALANCE_ENQUIRY (Account scouting)
       |
       v
TRANSFER_STARTED
       |
       v
BENEFICIARY_ENTERED (Unfamiliar NUBAN account number)
       |
       v
AMOUNT_ENTERED (Near maximum daily limit: e.g., ₦100,000)
       |
       v
PIN_ENTERED
```

Attackers habitually execute a **balance check** immediately before a transfer to verify available liquid funds. In contrast, regular legitimate users frequently proceed directly to transfer without scouting balance.

### 3.4 Historical USSD Customer Baselines

Customer baselines maintain channel-specific metrics:
- Typical USSD transfer amounts: $\mu_{\text{ussd}}, \sigma_{\text{ussd}}$.
- Habitual destination banks and recipient NUBANs.
- Typical session initiation hours (e.g., standard daylight hours vs. 3:00 AM).
- Frequency of USSD usage per day and week.

---

## 4. USSD-Native Intent Probes

In mobile banking apps, an intent probe can render rich modal dialogs. In USSD, the entire interface is constrained to **182 alphanumeric characters** per network screen.

When Parallax evaluates a transaction with moderate risk and high uncertainty (e.g., known SIM and valid PIN, but an urgent transfer to an unfamiliar recipient at an unusual hour), the policy engine emits `ActionProbe`.

The bank aggregator intercepts the authorization and presents a **USSD Intent Probe Screen** before the transfer is committed:

```text
+------------------------------------+
| Parallax Security Check:           |
| Purpose of this transfer?          |
| 1. Pay family/trusted merchant     |
| 2. Bank staff told me to send it   |
| 3. Caller claims urgent emergency  |
| 0. Cancel transfer                 |
+------------------------------------+
```

### 4.1 Response Processing

- **Input = 1 (Commercial / Legitimate)**: Resolves uncertainty toward legitimate. If within allowable limits, session completes.
- **Input = 2 (Bank Impersonation)**: Provides irrefutable context that the customer is actively being defrauded under false authority.
  - $P(\text{SocialEngineering})$ updates to $\ge 0.95$.
  - Parallax issues an immediate `BLOCK`.
  - The engine flags the recipient account across the interbank settlement network as a mule account.
  - An automated voice alert or SMS is dispatched to the customer explaining the impersonation tactic.
- **Input = 3 (Urgent Coercion)**: Issues `VERIFY / ESCALATE`, placing a temporary 4-hour cooling-off window on the transfer.
- **Input = 0 (Cancellation)**: Aborts the transfer safely with zero financial loss.

---

## 5. Latency and Timeout Budgeting

Carrier USSD gateways impose a hard 20 to 30 second timeout per dialog turn. To ensure zero session drops:

```text
+-------------------------------------------------------------+
| TOTAL ROUND-TRIP BUDGET: 20,000 ms                          |
+-------------------------------------------------------------+
| Carrier Radio Transmission (Handset -> Gateway):  3,000 ms  |
| Bank Aggregator Transport & Routing:                200 ms  |
| Parallax Feature Extraction & ML Inference:          12 ms  |
| Core Banking System Reservation:                    250 ms  |
| User Keypad Consideration & Input:               12,000 ms  |
| Carrier Radio Return Transmission:                3,000 ms  |
+-------------------------------------------------------------+
| Margin of Safety: 1,538 ms                                  |
+-------------------------------------------------------------+
```

Because Parallax executes GBDT inference in under **15 milliseconds**, the security layer consumes less than **0.1%** of the carrier's timeout window.

---

## 6. Offline and Telco Gateway Resiliency

Telco SS7 and SMPP links in emerging markets suffer from periodic packet drops and network instability. Parallax guarantees operational continuity via:

1. **Write-Ahead Log (WAL) Buffering**: If connectivity to the persistent data warehouse drops, USSD session telemetry is buffered locally in-memory and replayed upon reconnect.
2. **Local In-Process Inference**: The decision engine does not perform synchronous external cloud calls or remote LLM lookups during authorization.
3. **Fail-Secure Fallback**: If telco SIM swap verification APIs become completely unresponsive, Parallax gracefully falls back to local behavioral baselines and keystroke dynamics rather than halting payment services.
