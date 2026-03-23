import mysql.connector, jwt

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
win count (Integer)
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
		payload = jwt.decode(token, key, algorithms = ["Argon2"])
		username = payload["username"]
		email = payload["email"]
	except:
		raise TokenError("There was a problem accessing the user token")
	else:
		return username, email

def connect():
	"""
	Starts a connection with the database
	Outputs the database and a cursor to communicate with the database
	The connection is closed at the end of every function
	"""
	try:
		db = mysql.connector.connect(host = "localhost", port = "3307", user = "root", password = "pass", database = "mgt_db")
		cursor = db.cursor()
	except:
		raise ConnectionError("There was a problem connecting to the database")
	else:
		return db, cursor

def addPrompt(new_prompt):
	"""
	Takes in a prompt (String)
	Adds a new prompt to the prompts table if unique
	"""
	if not searchPrompt("prompt", new_prompt):
		db, cursor = connect()
		try:
			sql = "INSERT INTO prompts (prompt) VALUES (%s)"
			val = (new_prompt)
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
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")
	else:
		return result

def searchPrompt(attribute, value):
	"""
	Takes in an attribute (String) and value
	Data type of value must be correct to the attribute
	Outputs the first matching record (Tuple) found or None if there is no matching record
	"""
	db, cursor = connect()
	try:
		sql = "SELECT * FROM prompts WHERE " + attribute + " = %s"
		val = (value)
		cursor.execute(sql, val)
		result = cursor.fetchone()
		cursor.close()
		db.close()
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")
	else:
		return result

def removePrompt(attribute, value):
	"""
	Takes in an attribute (String) and value
	Data type of value must be correct to the attribute
	Removes the matching record from the prompts table
	"""
	db, cursor = connect()
	try:
		sql = "DELETE FROM prompts WHERE " + attribute	+ " = (%s)"
		val = (value)
		cursor.execute(sql, val)
		db.commit()
		cursor.close()
		db.close()
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")

def updatePrompt(prompt_id, new_prompt):
	"""
	Takes in the id (Integer) and prompt (String)
	Replaces the old prompt with the new prompt
	"""
	db, cursor = connect()
	if not searchPrompt("id", prompt_id):
		raise AccessError("This prompt does not exist")
	else:
		try:
			sql = "UPDATE prompts SET prompt = (%s) WHERE id = (%s)"
			val = (new_prompt, prompt_id)
			cursor.execute(sql, val)
			db.commit()
			cursor.close()
			db.close()
		except:
			cursor.close()
			db.close()
			raise AccessError("There was a problem accessing the database")

def addUser(token, key):
	"""
	Takes in a user token and a key
	Decodes the token using the key to get the username and email
	Adds a new user to the users table if the username is unique
	"""
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
	"""
	Outputs all user accounts in the users table
	"""
	db, cursor = connect()
	try:
		cursor.execute("SELECT * FROM users")
		result = cursor.fetchall()
		cursor.close()
		db.close()
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")
	else:
		return result

def searchUserWithToken(token, key):
	"""
	Takes in a token and key
	Outputs the first matching record (Tuple) with the username obtained from the token
	"""
	username, email = getUserDetails(token, key)
	return searchUserWithToken("username", username)

def searchUserWithoutToken(attribute, value):
	"""
	Takes in an attribute (String) and a value (String) for that attribute
	Outputs the first matching record (Tuple) or None if there is no matching record
	Avoid using email, win count and balance as they are not unique
	"""
	db, cursor = connect()
	try:
		sql = "SELECT * FROM users WHERE " + attribute + " = %s"
		val = (value)
		cursor.execute(sql, val)
		result = cursor.fetchone()
		cursor.close()
		db.close()
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")
	else:
		return result

def removeUser(attribute, value):
	"""
	Takes in an attribute (String) and value (String)
	Removes the matching record from the users table
	"""
	db, cursor = connect()
	try:
		sql = "DELETE FROM users WHERE " + attribute + " = (%s)"
		val = (value)
		cursor.execute(sql, val)
		db.commit()
		cursor.close()
		db.close()
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")

def updateUser(username, attribute, value):
	"""
	Takes in a username (String), attribute (String) and value
	Data type of value must be correct to the attribute
	Replaces the old value of the attribute of the user with the new value
	"""
	db, cursor = connect()
	if not searchUser("username", payload_username):
		raise AccessError("This user account does not exist")
	else:
		try:
			sql = "UPDATE users SET " + attribute + " = (%s) WHERE username = (%s)"
			val = (value, username)
			cursor.execute(sql, val)
			db.commit()
			cursor.close()
			db.close()
		except:
			cursor.close()
			db.close()
			raise AccessError("There was a problem accessing the database")

def deletePromptsTable():
	"""
	Deletes the prompts table
	"""
	db, cursor = connect()
	try:
		sql = "DROP TABLE IF EXISTS prompts"
		cursor.execute(sql)
		db.commit()
		cursor.close()
		db.close()
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")

def deleteUsersTable():
	"""
	Deletes the users table
	"""
	db, cursor = connect()
	try:
		sql = "DROP TABLE IF EXISTS users"
		cursor.execute(sql)
		db.commit()
		cursor.close()
		db.close()
	except:
		cursor.close()
		db.close()
		raise AccessError("There was a problem accessing the database")