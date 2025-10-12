import { useState, useRef } from 'react'
import { useNavigate, Link } from "react-router";
import css from './css/login.module.scss'

function Signup() {
    let navigate = useNavigate()

    const [status, setStatus] = useState("")
    const [loading, setLoading] = useState(false)

    const usernameRef = useRef<HTMLInputElement>(null)
    const passwordRef = useRef<HTMLInputElement>(null)

    async function handleLogin(e: React.MouseEvent<HTMLButtonElement>) {
        e.preventDefault()
        setLoading(true)
        const res = await fetch('/api/user/', {
            method: 'POST',
            headers: {
            'Content-Type': 'application/json'
            },
            body: JSON.stringify({
            username: usernameRef.current!.value, password: passwordRef.current!.value
            })
        })
        if (res.status == 200) {
            navigate("/home")
        }
        if (res.status == 409) {
            setStatus("This username is already taken.")
        }
        if (res.status == 500) {
            setStatus("Unknown server error occurred.")
        }
        if (res.status == 400 || res.status == 413) {
            setStatus("Given credentials are too long or empty.")
        }
        if (res.status != 200) {
            setLoading(false)
        }
    }

    return (
    <>
    <div className={css.wrapper}>
        <div className={css.container}>
            <h1 className={css.loginText}>Sign Up</h1>
            <form className={css.form}>
            <span className={css.inputsBox}>
                <input className={css.input} ref={usernameRef} type="text" 
                placeholder="Username" autoComplete="off" maxLength={25}/>
                <input className={css.input} ref={passwordRef} type="password" 
                placeholder="Password" maxLength={60}/>
            </span>
            <span className={css.statusText}>{status}</span>
            <button disabled={loading} className={!loading ? css.loginButton : css.loginButtonBlocked} 
            onClick={handleLogin}>
                <span>Sign Up</span>
            </button>			
            </form>
        </div>
        <span className={css.footer}>Or log in <Link to="/login" className={css.link}>here</Link>
        , if you already have an account.
        </span>
    </div>
    </>
    )
}

export default Signup
