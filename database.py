# app/database.py
import os
import mysql.connector
import jwt
from app.config import settings

# NOTE: Credentials for the REMOTE database NOT LOCAL (in .env)
DB_HOST = os.getenv("DB_HOST")
DB_USER = os.getenv("DB_USER")
DB_PASS = os.getenv("DB_PASS")
DB_NAME = os.getenv("DB_NAME")

"""
Prompts Table
Attributes:
id (Integer) (Primary Key) (Unique)
prompt (String) (Unique)

Users Table
Attributes:
id (Integer) (Primary Key) (Unique)
username (String) (Unique)
email (String)
win_count (Integer)
balance (Integer)
"""

# Class that represents problems encountered while connecting to the database
class ConnectionError(Exception):
    def __init__(self, message):
        self.message = message
        super().__init__(self.message)

# Class that represents problems encountered while accessing the database
class AccessError(Exception):
    def __init__(self, message):
        self.message = message
        super().__init__(self.message)

# Class that represents problems encountered while accessing a token
class TokenError(Exception):
    def __init__(self, message):
        self.message = message
        super().__init__(self.message)

def getUserDetails(token, key):
    """
    Takes in a token and key
    Outputs the username (String) and email (String) of the account
    """
    try:
        # NOTE: Changed algorithm from "Argon2" to HS256 to match standard JWT usage
        payload = jwt.decode(token, key, algorithms = ["HS256"])
        username = payload.get("username")
        email = payload.get("email")
        return username, email
    except Exception as e:
        raise TokenError(f"There was a problem accessing the user token: {e}")

def connect():
    """
    Starts a connection with the database
    Outputs the database and a cursor to communicate with the database
    """
    try:
        db = mysql.connector.connect(
            host = DB_HOST,
            user = DB_USER,
            password = DB_PASS,
            database = DB_NAME,
        )
        cursor = db.cursor(dictionary = True) # Use dictionary=True for easier access
        return db, cursor
    except Exception as e:
        raise ConnectionError(f"There was a problem connecting to the database: {e}")

def addPrompt(new_prompt):
    """
    Takes in a prompt (String)
    Adds a new prompt to the prompts table if unique
    """
    if not searchPrompt("prompt", new_prompt):
        db, cursor = connect()
        try:
            sql = "INSERT INTO prompts (prompt) VALUES (%s)"
            val = (new_prompt,)
            cursor.execute(sql, val)
            db.commit()
            cursor.close()
            db.close()
        except:
            cursor.close()
            db.close()
            raise AccessError("There was a problem accessing the database")
    else:
        raise AccessError("This prompt already exists")

def getAllPrompts():
    """
    Outputs all prompts in the prompts table
    """
    db, cursor = connect()
    try:
        cursor.execute("SELECT * FROM prompts")
        result = cursor.fetchall()
        cursor.close()
        db.close()
        return result
    except:
        cursor.close()
        db.close()
        raise AccessError("There was a problem accessing the database")

def searchPrompt(attribute, value):
    """
    Takes in an attribute (String) and value
    Outputs the first matching record (Dict) found or None
    """
    db, cursor = connect()
    try:
        sql = f"SELECT * FROM prompts WHERE {attribute} = %s"
        val = (value,)
        cursor.execute(sql, val)
        result = cursor.fetchone()
        cursor.close()
        db.close()
        return result
    except:
        cursor.close()
        db.close()
        raise AccessError("There was a problem accessing the database")

def removePrompt(attribute, value):
    db, cursor = connect()
    try:
        sql = f"DELETE FROM prompts WHERE {attribute} = %s"
        val = (value,)
        cursor.execute(sql, val)
        db.commit()
        cursor.close()
        db.close()
    except:
        cursor.close()
        db.close()
        raise AccessError("There was a problem accessing the database")

def updatePrompt(prompt_id, new_prompt):
    db, cursor = connect()
    if not searchPrompt("id", prompt_id):
        raise AccessError("This prompt does not exist")
    else:
        try:
            sql = "UPDATE prompts SET prompt = %s WHERE id = %s"
            val = (new_prompt, prompt_id)
            cursor.execute(sql, val)
            db.commit()
            cursor.close()
            db.close()
        except:
            cursor.close()
            db.close()
            raise AccessError("There was a problem accessing the database")

# (Users table functions follow your provided logic)
def addUser(token, key):
    payload_username, payload_email = getUserDetails(token, key)
    db, cursor = connect()
    if not searchUserWithoutToken("username", payload_username):
        try:
            sql = "INSERT INTO users (username, email, win_count, balance) VALUES (%s, %s, %s, %s)"
            val = (payload_username, payload_email, 0, 0)
            cursor.execute(sql, val)
            db.commit()
            cursor.close()
            db.close()
        except:
            cursor.close()
            db.close()
            raise AccessError("There was a problem accessing the database")
    else:
        raise AccessError("This user account already exists")

def getAllUsers():
    db, cursor = connect()
    try:
        cursor.execute("SELECT * FROM users")
        result = cursor.fetchall()
        cursor.close()
        db.close()
        return result
    except:
        cursor.close()
        db.close()
        raise AccessError("There was a problem accessing the database")

def searchUserWithoutToken(attribute, value):
    db, cursor = connect()
    try:
        sql = f"SELECT * FROM users WHERE {attribute} = %s"
        val = (value,)
        cursor.execute(sql, val)
        result = cursor.fetchone()
        cursor.close()
        db.close()
        return result
    except:
        cursor.close()
        db.close()
        raise AccessError("There was a problem accessing the database")

def removeUser(attribute, value):
    db, cursor = connect()
    try:
        sql = f"DELETE FROM users WHERE {attribute} = %s"
        val = (value,)
        cursor.execute(sql, val)
        db.commit()
        cursor.close()
        db.close()
    except:
        cursor.close()
        db.close()
        raise AccessError("There was a problem accessing the database")

def updateUser(username, attribute, value):
    db, cursor = connect()
    if not searchUserWithoutToken("username", username):
        raise AccessError("This user account does not exist")
    else:
        try:
            sql = f"UPDATE users SET {attribute} = %s WHERE username = %s"
            val = (value, username)
            cursor.execute(sql, val)
            db.commit()
            cursor.close()
            db.close()
        except:
            cursor.close()
            db.close()
            raise AccessError("There was a problem accessing the database")