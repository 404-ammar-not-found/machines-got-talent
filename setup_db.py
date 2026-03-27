import mysql.connector
import sys

def setup():
    print("--- Machines Got Talent: Database Setup ---")
    try:
        # 1. Initial connection to MySQL (without selecting a DB)
        # XAMPP Defaults: Host=localhost, User=root, Password="", Port=3306
        db = mysql.connector.connect(
            host="localhost",
            user="root",
            password="",
            port="3306"
        )
        cursor = db.cursor()

        # 2. Create the main database
        print("[1/4] Creating database 'mgt_db'...")
        cursor.execute("CREATE DATABASE IF NOT EXISTS mgt_db")
        cursor.execute("USE mgt_db")

        # 3. Create the 'prompts' table (for AI failsafe jokes)
        print("[2/4] Creating 'prompts' table...")
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS prompts (
                id INT AUTO_INCREMENT PRIMARY KEY,
                prompt TEXT NOT NULL
            )
        """)

        # 4. Create the 'users' table (for persistent stats/economy)
        print("[3/4] Creating 'users' table...")
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS users (
                id VARCHAR(255) PRIMARY KEY,
                username VARCHAR(255) UNIQUE NOT NULL,
                email VARCHAR(255) UNIQUE NOT NULL,
                password_hash VARCHAR(255) NOT NULL,
                win_count INT DEFAULT 0,
                balance INT DEFAULT 0
            )
        """)

        # 5. Seed initial "Failsafe" prompts
        print("[4/4] Seeding starter prompts...")
        starter_prompts = [
            ("Why did the AI cross the road? To optimize the path to the other side!"),
            ("A robot walks into a bar... and asks for a byte."),
            ("What do you call a comedic computer? A LOL-gorithm!"),
            ("My AI told me a joke, but it was a bit bit-ter."),
            ("Knock knock. Who's there? Java. Java who? Java nice day!"),
            ("I asked an AI to write a joke about paper. It said it was 'tear-able'."),
            ("Why was the computer cold? It left its Windows open!")
        ]

        # Only insert if the table is currently empty
        cursor.execute("SELECT COUNT(*) FROM prompts")
        if cursor.fetchone()[0] == 0:
            cursor.executemany("INSERT INTO prompts (prompt) VALUES (%s)", [(p,) for p in starter_prompts])
            print(f"      Successfully added {len(starter_prompts)} starter prompts.")
        else:
            print("      Prompts table already contains data. Skipping seeding.")

        db.commit()
        cursor.close()
        db.close()
        print("\nSUCCESS: Database 'mgt_db' is fully configured and ready for use!")

    except mysql.connector.Error as err:
        print(f"\nCRITICAL ERROR: {err}")
        print("\nTROUBLESHOOTING:")
        print("1. Ensure XAMPP is open and the MySQL module is 'Running'.")
        print("2. Ensure MySQL is using Port 3306 (the default).")
        print("3. Ensure the 'root' user has no password (the default).")
        sys.exit(1)

if __name__ == "__main__":
    setup()
