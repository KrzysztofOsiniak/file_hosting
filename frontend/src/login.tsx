import { useRef } from 'react'
import css from './css/login.module.scss'

function Login() {
    const usernameRef = useRef<HTMLInputElement>(null)
    const passwordRef = useRef<HTMLInputElement>(null)

    async function handleLogin(e: React.MouseEvent<HTMLButtonElement>) {
    e.preventDefault()
    fetch('/api/user/login', {
        method: 'POST',
        headers: {
        'Content-Type': 'application/json'
        },
        body: JSON.stringify({
        username: usernameRef.current!.value, password: passwordRef.current!.value
        })
    })
    }

    return (
    <>
    <div className={css.container}>
        <h1 className={css.loginText}>Log In</h1>
        <form className={css.form}>
        <span className={css.inputsBox}>
            <input className={css.input} ref={usernameRef} type="text" 
            placeholder="Username" autoComplete="off" maxLength={25}/>
            <input className={css.input} ref={passwordRef} type="password" 
            placeholder="Password" maxLength={60}/>
        </span>
        <span className={css.statusText}></span>
        <button className={css.loginButton} onClick={handleLogin}>
            <span>Log In</span>
        </button>				
        </form>
    </div>
    <span className={css.footer}>Or create an account <span className={css.link}>here</span>, no email needed.</span>
    </>
    )
}

export default Login
