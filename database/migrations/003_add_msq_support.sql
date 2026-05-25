-- Migration: Add MSQ support and question types
ALTER TABLE questions ADD COLUMN type TEXT NOT NULL DEFAULT 'MCQ';
ALTER TABLE questions ALTER COLUMN correct_answer TYPE TEXT[] USING array[correct_answer];
ALTER TABLE questions RENAME COLUMN correct_answer TO correct_answers;

-- Update quiz_answers to handle multiple selected answers
ALTER TABLE quiz_answers ALTER COLUMN selected_answer TYPE TEXT[] USING array[selected_answer];
ALTER TABLE quiz_answers RENAME COLUMN selected_answer TO selected_answers;
