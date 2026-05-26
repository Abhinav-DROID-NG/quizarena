# Elo Algorithm Notes

QuizArena updates user Elo after each answer using:

1. **Expected probability**
   \[
   E = \frac{1}{1 + 10^{(Q-U)/400}}
   \]
   where `U` is user Elo and `Q` is question Elo.

2. **Performance score**
   - 80% timing component (`time_score`)
   - 20% correctness component

3. **Difficulty K-factor**
   - easy: 16
   - medium: 24
   - hard: 32

4. **Anti-guessing penalty**
   Fast incorrect responses and repeated skips reduce Elo gains/increase losses.

5. **Clamping**
   Final Elo is clamped into `[0, 5000]`.

Implementation reference: `services/elo.go` and handler usage in `handlers/quiz.go`.
