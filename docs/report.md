# Financial Track — Intent-Aware Fraud Detection

Background research and problem framing that motivated Parallax's design. This
document predates the implementation; it explains *why* intent inference was chosen
over conventional anomaly-score fraud detection, and why an LLM-as-classifier
architecture was rejected. See the [README](../README.md) for the system as built.

## 1. Executive Summary

The Financial Services & Digital Payments challenge asks participants to detect
account takeover from user behaviour during live financial transactions. The
conventional interpretation is to build a behavioural anomaly detector: establish a
customer's normal behaviour, observe a transaction, calculate a risk score, and block
or challenge the transaction when the score becomes sufficiently high.

Our investigation suggests that this framing is incomplete.

The fundamental problem is not simply that financial systems fail to detect unusual
behaviour. It is that financial systems observe the traces of a person's actions but
have very limited visibility into the intent behind those actions.

A user can authenticate successfully, operate from a familiar device, enter the
correct PIN and OTP, and still perform a transaction that they did not genuinely
intend in the ordinary sense. This can happen because the account has been
compromised, because the user was socially engineered, because they made an
accidental transfer, or because they were coerced or manipulated.

The same observable transaction can therefore correspond to radically different
underlying states:

- legitimate and intentional
- accidental but authenticated
- socially engineered
- account takeover
- potentially coerced or otherwise ambiguous

This makes a simple fraud score insufficient.

The proposed direction is an intent-inference security layer positioned around the
payment flow. Rather than asking only whether a transaction is anomalous, the system
estimates competing explanations for the transaction, evaluates the uncertainty
surrounding those explanations, and collects additional evidence when the system
cannot confidently establish the user's intent.

The central engineering principle is:

> Authentication establishes that an action was authorized by credentials. Behaviour
> provides evidence about how the action occurred. Neither alone establishes why the
> action occurred.

The system therefore treats intent as a latent variable that must be inferred from
multiple forms of evidence rather than directly "read" from raw transaction data.

## 2. The Problem

### 2.1 The limitation of conventional transaction fraud detection

Most conventional financial security controls operate around questions such as:

> Was the correct password entered? Was the OTP valid? Is the device known? Is the
> transaction amount unusually large? Is the recipient new? Is the location unusual?
> Has this account behaved this way previously?

These are useful signals, but they largely answer:

> "Does this activity look unusual or authenticated?"

They do not necessarily answer:

> "Did the person actually intend to perform this action?"

This distinction becomes critical in modern financial fraud.

Consider a transaction of ₦400,000, with:

- Correct credentials
- Correct OTP
- Known account
- Valid device
- Successful authorization

The transaction can still represent legitimate payment, an accidental transfer,
social engineering, account takeover, or coercion / another abnormal circumstance.

The infrastructure sees almost identical technical evidence in each case.

The meaningful variable — the user's actual intent — remains largely hidden.

## 3. Why Intent Matters

A useful conceptual model is:

```text
Actual Intent
     ↓
Psychological / Situational State
     ↓
Decision-making
     ↓
User Behaviour
     ↓
Observable System Signals
```

Existing financial systems predominantly observe the final layer.

The proposed research direction moves the problem upward.

Instead of treating observable behaviour as the final answer, the system asks:

> What underlying intent is most consistent with the evidence we have?

This makes fraud detection an inference problem under uncertainty rather than merely
an anomaly-detection problem.

Importantly, the objective is not to claim that software can directly read a user's
mind. The objective is to determine whether the available evidence is sufficient to
distinguish plausible explanations for an action.

## 4. Findings From Real-World Discussions

Reddit was examined as a source of real-world incident narratives and user
experiences. These discussions are anecdotal and cannot establish prevalence, but
they are valuable for identifying failure modes that users actually encounter.

The findings were also compared with official Nigerian financial-sector guidance,
particularly Central Bank of Nigeria material concerning fraud, scams, consumer
complaints, and authorized push payment fraud.

### 4.1 Authentication can succeed while an account is still compromised

A recent discussion from fraud professionals described account takeovers where
established customers experienced clean logins, sometimes from recognized devices,
and successfully passed MFA. The suspicious activity occurred after authentication,
including changes to payout information followed by money movement. The poster
specifically argued that the important signals occurred after the "front door"
authentication event.

This directly challenges the assumption that successful authentication provides
sufficient evidence of legitimacy.

A separate 2026 account-takeover report described an attacker changing the
customer's phone number, circumventing MFA, and subsequently taking control of the
account.

**Implication.** The security boundary cannot end at login. The relevant unit of
analysis may need to be the entire authenticated session and its sequence of state
changes.

### 4.2 Social engineering changes intent without necessarily changing authentication

Several discussions illustrate a more difficult class of fraud: the customer
performs the action themselves because they believe they are following legitimate
instructions.

In one 2026 case, scammers impersonated a bank's fraud department, created a false
sense of urgency, claimed to be investigating fraudulent activity, and persuaded the
victim to provide a verification code and move money as part of the supposed
security process.

Another account described an attempted OTP scam in which the attacker rapidly
delivered specific transaction information while impersonating the bank's fraud
team. The victim described being in the middle of another activity when the call
occurred and nearly complied.

The Central Bank of Nigeria independently identifies social engineering as a fraud
mechanism in which criminals exploit trust, fear, authority, and urgency to
manipulate individuals into revealing confidential information or taking harmful
actions.

The CBN's draft guidelines on authorized push payment fraud are particularly
relevant: they explicitly describe fraud that uses social engineering to exploit
customer trust and the finality of digital transactions, and they frame prevention,
detection, reporting and resolution of APP fraud as a distinct financial-system
concern.

**Implication.** A transaction can be technically authorized but psychologically
manipulated. This is a major gap between authentication and actual intent.

## 5. Accidental Transactions Create a Different Kind of Ambiguity

Not every harmful transaction is malicious.

A 2026 Nigerian Reddit discussion focused on mistaken bank transfers and described
situations involving wrong beneficiaries, incorrect amounts and large accidental
transfers. One commenter reported mistakenly transferring almost ₦1 million, while
another described an unresolved ₦350,000 mistaken transfer. The discussion also
highlighted late-night usage, fatigue and hurried interaction as possible
contributors to errors.

The discussion is user-generated rather than independently validated research, so
the numerical claims in that post should not be treated as population statistics.
Nevertheless, the underlying failure mode is important:

> A user can legitimately authenticate and intentionally press "Send" while still
> not intending the resulting financial outcome.

For example:

```text
Intended: ₦10,000  → Beneficiary A
Actual:   ₦100,000 → Beneficiary B
```

The banking system may correctly record that the user authenticated, the
transaction was confirmed, and the transfer completed — but those facts do not mean
the user intended that exact financial outcome.

**Implication.** A robust system must distinguish malicious abnormality from human
error. Otherwise, attempts to improve fraud detection may simply create more false
positives.

## 6. Unusual Behaviour Is Not Equivalent to Fraud

The challenge itself correctly notes that people share phones, change SIM cards, use
different channels, and may operate under poor connectivity.

Real-world discussions reinforce this broader problem.

- A **new device** might mean an attacker, a broken old phone, a new phone purchase,
  or a shared family device.
- A **large transfer** might mean fraud, rent, school fees, a medical expense, a
  business payment, or a family obligation.
- A **transaction at 1 AM** might mean fraud, a night-shift worker, travel, an
  emergency, or simply someone who banks at night.
- A **network-related retry pattern** might mean an attack, or a user repeatedly
  retrying a failed transaction.

The consequence is fundamental:

> Anomaly is evidence, not a verdict.

A security system that equates deviation from baseline with fraud will inevitably
punish legitimate customers.

## 7. Network and Channel Constraints Matter

Nigerian digital banking operates across multiple channels, including mobile apps,
internet banking, USSD, POS and ATM infrastructure.

These channels expose radically different amounts of behavioural information.

A smartphone session might provide typing timing, touch interactions, device
characteristics, navigation sequence, session duration, clipboard interaction, and
application state.

A USSD session may provide little more than:

```text
Menu → option → option → amount → confirmation
```

Feature-phone users may provide even less.

Network instability compounds the problem. Users may retry transactions, switch
channels or receive delayed notifications.

Therefore:

> The absence of rich telemetry cannot be interpreted as evidence of malicious
> behaviour.

The system must remain meaningful when the richest behavioural signals are
unavailable.

## 8. Post-Transaction Detection Is Often Too Late

Fraud becomes increasingly difficult to contain as time passes.

The security timeline is therefore important:

```text
Account compromise
       ↓
Suspicious interaction
       ↓
Transaction initiated
       ↓
Transaction authorized
       ↓
Funds moved
       ↓
Money redistributed
       ↓
Customer notices
       ↓
Complaint
       ↓
Investigation
       ↓
Recovery
```

The earlier uncertainty can be detected, the more options exist.

The later it is discovered, the problem changes from "Should this transaction
happen?" to "Can the transaction be reversed?" to "Can the money be recovered?"

The CBN's consumer complaint process reflects the seriousness of this downstream
problem: customers are expected to report disputes to their financial institution
first and can escalate unresolved complaints to the CBN's Consumer Protection
Department.

This suggests that fraud prevention cannot be treated as an isolated classification
problem. It is part of a broader incident lifecycle.

## 9. Human Psychology Is Part of the Attack Surface

The observed cases consistently point to psychological mechanisms:

- **Urgency** — the attacker manufactures a short decision window.
- **Authority** — the attacker impersonates a bank, fraud team, employer, government
  official or another trusted actor.
- **Fear** — the user believes something bad has already happened and is attempting
  to "fix" it.
- **Cognitive overload** — the victim receives information rapidly and has less
  opportunity to independently evaluate the request.
- **Trust** — the attacker inserts themselves into an existing trusted relationship.
- **Commitment** — after taking one action, the person becomes progressively
  invested in continuing.
- **Self-confidence** — a person may believe they are too technically aware or
  intelligent to be scammed, potentially reducing their skepticism.

The important engineering observation is not that software should attempt to
diagnose a user's emotions.

It is that financial risk may depend on the state in which the decision was made.
That state can leave observable traces.

## 10. The Core Technical Problem

The resulting problem can be formulated as:

> Given an authenticated financial session, determine whether the observed action is
> most consistent with legitimate intent, accidental intent, social manipulation,
> account compromise, or another abnormal state, while explicitly representing
> uncertainty and minimizing false positives.

Formally, the system can treat intent as a hidden variable:

```text
Observed Evidence
        ↓
P(Intent | Evidence)
```

where intent represents competing hypotheses rather than a binary label.

For example:

```text
Legitimate             0.72
Accidental             0.09
Social engineering     0.15
Account takeover       0.04
```

These values should not simply be generated by a language model.

They should emerge from an engineered evidence pipeline whose components can be
measured, tested and independently evaluated.

## 11. Why "Send Everything to an LLM and Ask for a Risk Score" Is Weak Engineering

An architecture such as:

```text
Transaction JSON
      ↓
     LLM
      ↓
"Risk = 0.83"
```

is attractive for a prototype but problematic as the primary security mechanism.

The main issues include:

- **Reproducibility** — the same underlying case can potentially produce different
  natural-language reasoning or decisions.
- **Calibration** — a natural-language model's numerical confidence is not
  automatically a calibrated probability.
- **Evidence contamination** — irrelevant or poorly structured contextual
  information can influence the output.
- **Explainability after the fact** — the model may produce a persuasive explanation
  without that explanation corresponding to a real, measurable decision boundary.
- **Testing** — it is difficult to establish rigorous performance guarantees when
  the central decision mechanism is unconstrained language generation.
- **Security** — a critical financial decision should not depend on a component
  that is allowed to infer arbitrary conclusions from loosely structured input.

**Engineering boundary.** A security system should distinguish evidence collection,
from risk calculation, from policy decision, from human-readable explanation.

An LLM can contribute to reasoning or explanation without becoming the sole
authority that decides whether money moves.

## 12. Proposed System Concept

The proposed system is therefore an intent-aware security layer around payment
sessions.

The conceptual architecture is:

```text
                 PAYMENT SESSION
                       │
                       ▼
                EVENT INGESTION
                       │
                       ▼
                FEATURE ENGINE
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   Behaviour       Sequence        Context
    Analysis        Analysis        Analysis
        │              │              │
        └──────────────┼──────────────┘
                       ▼
                EVIDENCE LAYER
                       │
                       ▼
              INTENT HYPOTHESES
                       │
                       ▼
             UNCERTAINTY ESTIMATION
                       │
            ┌──────────┴──────────┐
            ▼                     ▼
      High confidence        High uncertainty
            │                     │
            ▼                     ▼
         Continue             Intent Probe
                                  │
                                  ▼
                           New Evidence
                                  │
                                  ▼
                         Updated Hypotheses
                                  │
                                  ▼
                           POLICY ENGINE
                                  │
                 ┌────────────────┼────────────────┐
                 ▼                ▼                ▼
              Proceed          Verify           Block/
                                                 Escalate
```

The fundamental design choice is that uncertainty becomes a first-class output.

The system is not forced to claim "This is fraud" when the available evidence only
supports "We do not currently know what this transaction represents."

## 13. Competing Intent Hypotheses

Rather than a single binary classifier (`fraud / not fraud`), the system maintains a
set of competing explanations.

A prototype could include:

- `LEGITIMATE`
- `ACCIDENTAL`
- `SOCIAL_ENGINEERING`
- `ACCOUNT_TAKEOVER`
- `UNKNOWN / COERCED`

The actual category set can later be refined according to the dataset and evidence.

This produces a much richer output. For example:

```text
Account takeover:      0.07
Social engineering:    0.68
Accidental:            0.11
Legitimate:            0.14
```

The important result is not simply that the system believes the transaction is
dangerous. It is that the system has formed a hypothesis about *why* it may be
dangerous.

## 14. Evidence Should Be Structured Before AI Is Involved

A critical engineering boundary is the separation between raw telemetry and model
reasoning.

Instead of feeding a large JSON object directly to an LLM, the system should first
transform raw activity into explicit evidence. For example:

```json
{
  "amount_deviation": 7.4,
  "new_beneficiary": true,
  "beneficiary_age_seconds": 19,
  "device_known": true,
  "credential_changed_recently": false,
  "transaction_time_deviation": 3.1,
  "interaction_speed_deviation": 2.4,
  "recipient_seen_before": false
}
```

The evidence layer can then determine:

- Transaction amount is 7.4× historical baseline.
- Recipient has never been used before.
- Recipient was created 19 seconds before payment.
- Device itself is known.

Only then can an AI component reason over the structured evidence.

This means AI becomes a constrained investigator or reasoning assistant, rather than
the entire fraud system.

## 15. The System Should Reason About Sequences

A transaction should not necessarily be treated as a single record. It should be
viewed as an event sequence:

```text
LOGIN
  ↓
DEVICE_CHANGE
  ↓
PASSWORD_CHANGE
  ↓
BENEFICIARY_CREATE
  ↓
TRANSFER
```

versus:

```text
LOGIN
  ↓
KNOWN_BENEFICIARY
  ↓
TRANSFER
```

The second sequence is ordinary.

The first contains a series of state changes that collectively provide
substantially more information than the final transfer alone.

This enables a concept of causal or sequential evidence:

> What happened immediately before the transaction, and what relationship does that
> sequence have to the user's normal behaviour?

## 16. The System Should Distinguish Risk From Uncertainty

This distinction is central.

Two users can both produce transactions that deviate from their historical
behaviour.

- **Case A** — Risk: high, intent uncertainty: low. For example, several strong
  indicators point toward account compromise.
- **Case B** — Risk: moderate, intent uncertainty: high. The transaction is highly
  unusual, but legitimate explanations remain plausible.

These cases should not necessarily receive the same response.

This creates an additional decision dimension:

```text
                    INTENT CERTAINTY
                         ↑
                         │
        Proceed safely   │   Proceed cautiously
                         │
─────────────────────────┼────────────────────→
                         │             RISK
        Ask for context  │   Strong intervention
                         │
                         ↓
```

This is more expressive than a single threshold.

## 17. Intent Probing

When the available evidence cannot distinguish between plausible intent states, the
system can seek additional evidence.

This should not be implemented as an ineffective "Are you sure?" confirmation.

Instead, the interaction should attempt to expose the user's understanding of the
transaction — for example, "What is this payment for?" or another
context-appropriate question.

A user's answer becomes another structured evidence source. For example:

```text
Observed:
  New beneficiary
  Large amount
  Unusual session

User-stated purpose:
  "Bank told me to send this there so they can reverse a fraudulent transaction."
```

The system now has evidence that was not available from transaction metadata alone.

The important concept is:

> When passive observation is insufficient, deliberately acquire information that
> reduces uncertainty about intent.

## 18. Adaptive Friction

The system should not make every transaction harder.

A customer whose transaction is consistent with established behaviour should
experience essentially no change. A transaction with strong evidence of compromise
may be blocked or escalated. A transaction with ambiguous evidence may receive an
additional verification step.

Conceptually:

```text
Low risk + low uncertainty        → Proceed
Moderate risk + high uncertainty  → Gather additional evidence
High risk + strong evidence       → Block / escalate
```

This directly addresses the challenge requirement that legitimate transactions
should not be unnecessarily interrupted.

## 19. Why This Is Infrastructure-Heavy

The frontend is intentionally not the centre of the project.

The difficult engineering exists underneath:

```text
Event collection
       ↓
Event normalization
       ↓
Real-time feature extraction
       ↓
Per-user baseline storage
       ↓
Sequence analysis
       ↓
Model inference
       ↓
Intent hypothesis generation
       ↓
Uncertainty estimation
       ↓
Intervention policy
       ↓
Decision logging
```

The frontend only visualizes what the underlying system is doing.

A minimal prototype UI is sufficient:

```text
┌────────────────────────────────────────────┐
│ LIVE PAYMENT SESSION                       │
├────────────────────────────────────────────┤
│ Login                                      │
│ Beneficiary created                        │
│ Transfer initiated                         │
├────────────────────────────────────────────┤
│ INTENT HYPOTHESES                           │
│                                            │
│ Social engineering       68%               │
│ Legitimate               21%               │
│ Accidental                8%               │
│ Account takeover          3%               │
├────────────────────────────────────────────┤
│ KEY EVIDENCE                               │
│ • New beneficiary                          │
│ • 7.4× normal amount                       │
│ • Unusual interaction sequence             │
├────────────────────────────────────────────┤
│ DECISION                                   │
│ Additional intent verification required   │
└────────────────────────────────────────────┘
```

The visual interface communicates the result; the infrastructure produces it.

## 20. Demonstration Model

The strongest demonstration should show that the same final transaction can have
different underlying intent.

**Scenario 1 — Legitimate.** Known device, known beneficiary, normal amount, normal
time, normal interaction.

```text
Result: Legitimate 96% → Proceed
```

**Scenario 2 — Account takeover.** New device, credential modification, new
beneficiary, rapid post-login changes, large transfer.

```text
Result: Account takeover 87% → Block / escalate
```

**Scenario 3 — Social engineering.** Known device, correct credentials, valid OTP,
new beneficiary, large amount, abnormal interaction pattern. User explanation: "Bank
told me to move the money to secure it."

```text
Result: Social engineering 91% → Pause + intervention
```

The third scenario is particularly important because the transaction is technically
authenticated. It demonstrates the central hypothesis:

> Authentication does not establish intent.

## 21. Replayable Event Stream

The prototype should be capable of replaying synthetic sessions. For example:

```text
09:41:02  LOGIN
09:41:06  DEVICE_CHECK
09:41:11  BENEFICIARY_CREATE
09:41:24  AMOUNT_ENTER
09:41:38  TRANSFER_INITIATED
```

The inference engine consumes these events in real time.

The displayed intent distribution changes as evidence accumulates. For example:

```text
After LOGIN:            Legitimate 92%
After NEW BENEFICIARY:  Legitimate 61%, Social engineering 21%
After LARGE TRANSFER:   Legitimate 24%, Social engineering 61%
After USER EXPLANATION: Social engineering 91%
```

This demonstrates something important:

> Intent inference is incremental.

The system does not know everything at the beginning of the session. Evidence
accumulates.

## 22. Dataset and Evaluation

Because real personal financial data is prohibited by the challenge, the prototype
should use a synthetic transaction and session generator.

The generator should create multiple populations: legitimate users, account
takeover sessions, social engineering sessions, and accidental transactions.

Each synthetic customer should have a behavioural history rather than a random
collection of transactions. For example:

```text
Customer A
-----------------------------
Typical amount: ₦10k–₦75k
Typical payment time: 08:00–18:00
Typical beneficiaries: 6
Primary device: Android
Typical interaction duration: 25–50 sec
```

Synthetic attack scenarios can then alter specific dimensions.

The test set should measure:

- **Detection performance** — precision, recall, false-positive rate and
  false-negative rate.
- **Intent classification performance** — whether the system correctly
  differentiates legitimate, accidental, social engineering and account takeover.
- **Calibration** — whether confidence values correspond meaningfully to observed
  frequencies.
- **Intervention efficiency** — how often additional verification is invoked.
- **Customer impact** — how many legitimate transactions are unnecessarily
  interrupted.

The challenge explicitly requires showing both detection performance and
false-alarm behaviour, so false positives should be treated as a first-class
evaluation metric rather than an afterthought.

## 23. Comparison Baseline

An especially useful experimental design is to compare several approaches.

**Baseline 1 — Static transaction rules.**

```text
Large amount OR new beneficiary OR unusual time → flag
```

This provides a simple reference point.

**Baseline 2 — Behavioural anomaly detection.**

```text
User history → Anomaly score → Threshold
```

This tests the challenge's conventional behavioural approach.

**Proposed approach.**

```text
Behavioural evidence
+ Temporal sequence
+ Transaction context
+ Intent hypotheses
+ Uncertainty
+ Additional evidence when needed
```

The comparison allows the project to demonstrate not merely that the proposed
system works, but why a richer formulation of the problem is useful.

## 24. Important Limitations

The system should not claim to truly "know" a customer's intent.

Intent is inferred from incomplete evidence. A sophisticated attacker can
deliberately imitate normal behaviour. Behavioural patterns naturally change. People
share devices. USSD provides less telemetry than smartphones. Psychological state
cannot be reliably diagnosed from interaction traces alone. A customer can provide a
misleading explanation.

Therefore the system should explicitly retain evidence, hypothesis, confidence, and
uncertainty, rather than presenting its inference as absolute truth.

The correct security posture is:

> "Based on the evidence currently available, this is the most likely explanation."

not:

> "We know what the customer intended."

## 25. Broader Finding

The research ultimately points to a broader limitation in digital financial
security.

The industry has become increasingly sophisticated at answering "Who
authenticated?" and "Does this transaction resemble known fraud?"

The harder question is: "What is the human actually trying to accomplish?"

That question matters because modern financial fraud increasingly operates through
valid actions performed under invalid circumstances.

The criminal does not always need to bypass the bank. Sometimes they can make the
legitimate user become the mechanism through which the fraudulent transaction is
authorized.

This changes the security model. The system should therefore move from:

```text
Authentication + Anomaly Detection
```

toward:

```text
Authentication
+ Behavioural Evidence
+ Context
+ Intent Inference
+ Uncertainty
+ Adaptive Verification
```

The goal is not to eliminate uncertainty. The goal is to recognize uncertainty early
and respond proportionally.

## 26. Core Problem Statement

The clearest formulation emerging from the research is:

> Financial systems can observe what a user does, but they cannot reliably
> determine what the user intended to do.

A more technical formulation is:

> Design a real-time security layer that infers probable user intent from
> authenticated session behaviour and transaction context, distinguishes between
> legitimate, accidental, manipulated and compromised actions, explicitly models
> uncertainty, and gathers additional evidence when the available signals are
> insufficient.

That is the problem space the prototype should investigate.

It is deliberately broader than "Build an AI fraud detector."

It is instead: **build infrastructure for reasoning about intent in financial
transactions.**
