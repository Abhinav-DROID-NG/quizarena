CREATE INDEX IF NOT EXISTS idx_users_elo_id ON users(current_elo DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_questions_subject_elo_id ON questions(subject, question_elo, id);
CREATE INDEX IF NOT EXISTS idx_questions_elo_id ON questions(question_elo, id);
CREATE INDEX IF NOT EXISTS idx_sessions_subject_user ON quiz_sessions(subject, user_id);
CREATE INDEX IF NOT EXISTS idx_answers_session_created ON quiz_answers(session_id, created_at);
