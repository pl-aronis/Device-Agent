-- Add response_data column to store command response bodies (e.g., DeviceInformation results)
ALTER TABLE commands ADD COLUMN response_data TEXT;