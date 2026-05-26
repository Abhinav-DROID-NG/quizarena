CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  google_sub TEXT UNIQUE,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT,
  name TEXT NOT NULL,
  picture TEXT,
  current_elo INT NOT NULL DEFAULT 1200,
  peak_elo INT NOT NULL DEFAULT 1200,
  accuracy_percentage DOUBLE PRECISION NOT NULL DEFAULT 0,
  average_response_time DOUBLE PRECISION NOT NULL DEFAULT 0,
  total_questions_solved INT NOT NULL DEFAULT 0,
  strongest_subject TEXT NOT NULL DEFAULT '',
  weakest_subject TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS questions (
  id BIGSERIAL PRIMARY KEY,
  subject TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'MCQ',
  difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
  question_text TEXT NOT NULL,
  options TEXT[] NOT NULL,
  correct_answers TEXT[] NOT NULL,
  question_elo INT NOT NULL DEFAULT 1200,
  expected_time_seconds DOUBLE PRECISION NOT NULL DEFAULT 20,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quiz_sessions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quiz_answers (
  id BIGSERIAL PRIMARY KEY,
  session_id BIGINT NOT NULL REFERENCES quiz_sessions(id) ON DELETE CASCADE,
  question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  selected_answers TEXT[] NOT NULL,
  correct BOOLEAN NOT NULL,
  time_taken_seconds DOUBLE PRECISION NOT NULL,
  time_score DOUBLE PRECISION NOT NULL,
  performance_score DOUBLE PRECISION NOT NULL,
  elo_change INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_elo_id ON users(current_elo DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_questions_subject_elo_id ON questions(subject, question_elo, id);
CREATE INDEX IF NOT EXISTS idx_questions_elo_id ON questions(question_elo, id);
CREATE INDEX IF NOT EXISTS idx_sessions_user_subject ON quiz_sessions(user_id, subject);
CREATE INDEX IF NOT EXISTS idx_sessions_subject_user ON quiz_sessions(subject, user_id);
CREATE INDEX IF NOT EXISTS idx_answers_session_created ON quiz_answers(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_answers_user_created ON quiz_answers(user_id, created_at);
