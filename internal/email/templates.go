package email

const otpEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, Helvetica, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
    <div style="background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
        <h2 style="color: #333; margin-bottom: 20px;">Hi {{.Name}},</h2>

        <p style="color: #555; font-size: 16px; line-height: 1.6;">
            Your verification code is:
        </p>

        <div style="background: #f8f9fa; padding: 25px; text-align: center; border-radius: 8px; margin: 20px 0; border: 1px solid #e9ecef;">
            <span style="font-size: 36px; letter-spacing: 10px; font-weight: bold; color: #333; font-family: 'Courier New', monospace;">
                {{.OTP}}
            </span>
        </div>

        <div style="margin: 25px 0; padding: 15px; background: #e8f4f8; border-radius: 6px;">
            <p style="margin: 0; color: #555;"><strong>Purpose:</strong> {{.Purpose}}</p>
            <p style="margin: 5px 0 0 0; color: #555;"><strong>Expires in:</strong> 10 minutes</p>
        </div>

        <p style="color: #888; font-size: 14px; margin-top: 30px;">
            If you didn't request this email, you can safely ignore it. This message was sent to {{.Name}}.
        </p>
    </div>

    <div style="text-align: center; margin-top: 20px; color: #999; font-size: 12px;">
        <hr style="border: none; border-top: 1px solid #ddd; margin-bottom: 15px;">
        <p style="margin: 0;">— The PonyPigeon Team</p>
        <p style="margin: 5px 0 0 0;">no-reply@ponypigeon.com</p>
    </div>
</body>
</html>`
